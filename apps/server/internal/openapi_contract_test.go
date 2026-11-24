package internal_test

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go-starter/internal/kernel/audit"
	"go-starter/internal/kernel/storage"
	"go-starter/internal/modules/files"
	"go-starter/internal/modules/iam"
	"go-starter/internal/modules/sample"
)

type routeMeta struct {
	loc       string
	needsAuth bool
}

func TestOpenAPIContractMatchesRegisteredRoutesAndAuthLevel(t *testing.T) {
	serverRoot := filepath.Clean("..")
	registered, err := collectRegisteredHTTPRoutes(serverRoot)
	if err != nil {
		t.Fatalf("failed to scan Go HTTP routes: %v", err)
	}

	specPath := filepath.Join(serverRoot, "docs", "openapi.yaml")
	documented, err := parseOpenAPIRoutes(specPath)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", specPath, err)
	}

	var errs []string
	for route, meta := range registered {
		specNeedsAuth, exists := documented[route]
		if !exists {
			errs = append(errs, fmt.Sprintf("路由 %s (%s) 未写入 docs/openapi.yaml", route, meta.loc))
			continue
		}
		if meta.needsAuth != specNeedsAuth {
			errs = append(errs, fmt.Sprintf(
				"路由 %s (%s) 鉴权声明失步：Go needsAuth=%v，而 openapi.yaml BearerAuth=%v",
				route, meta.loc, meta.needsAuth, specNeedsAuth,
			))
		}
	}
	for route := range documented {
		if _, ok := registered[route]; !ok {
			errs = append(errs, fmt.Sprintf("docs/openapi.yaml 声明了未在 Go 代码中注册的幽灵路由：%s", route))
		}
	}
	slices.Sort(errs)
	if len(errs) > 0 {
		t.Errorf("[契约与鉴权漂移] 检测到 %d 处不一致：\n  - %s", len(errs), strings.Join(errs, "\n  - "))
	}
}

func TestOpenAPISchemasMatchGoStructs(t *testing.T) {
	specPath := filepath.Join("..", "docs", "openapi.yaml")
	specSchemas, err := parseOpenAPISchemas(specPath)
	if err != nil {
		t.Fatalf("parse openapi schemas: %v", err)
	}

	bindings := map[string]any{
		"TokenPair":           iam.TokenPair{},
		"User":                iam.User{},
		"Role":                iam.Role{},
		"Permission":          iam.Permission{},
		"MeResp":              iam.MeResp{},
		"SampleItem":          sample.Item{},
		"AuditEntry":          audit.Entry{},
		"FileInfo":            storage.FileInfo{},
		"MultipartSession":    storage.MultipartSession{},
		"LoginReq":            iam.LoginReq{},
		"RefreshReq":          iam.RefreshReq{},
		"ChangePasswordReq":   iam.ChangePasswordReq{},
		"CreateUserReq":       iam.CreateUserReq{},
		"UpdateUserStatusReq": iam.UpdateUserStatusReq{},
		"AssignUserRolesReq":  iam.AssignUserRolesReq{},
		"CreateRoleReq":       iam.CreateRoleReq{},
		"AssignRolePermsReq":  iam.AssignRolePermsReq{},
		"CreateSampleReq":     sample.CreateReq{},
		"InitMultipartReq":    files.InitMultipartReq{},
	}

	var mismatches []string
	for schemaName, sampleVal := range bindings {
		specDef, ok := specSchemas[schemaName]
		if !ok {
			mismatches = append(mismatches, fmt.Sprintf("openapi.yaml 缺少 components.schemas.%s", schemaName))
			continue
		}
		goProps, goReq, hasValidate := inspectGoStructJSON(reflect.TypeOf(sampleVal))

		for f := range goProps {
			if !specDef.props[f] {
				mismatches = append(mismatches, fmt.Sprintf("%s: Go 结构体字段 %q 未写入 openapi.yaml properties", schemaName, f))
			}
		}
		for f := range specDef.props {
			if !goProps[f] {
				mismatches = append(mismatches, fmt.Sprintf("%s: openapi.yaml 声明了 Go 结构体不存在的字段 %q", schemaName, f))
			}
		}
		if hasValidate {
			for f := range goReq {
				if !specDef.required[f] {
					mismatches = append(mismatches, fmt.Sprintf("%s: Go 必填字段 %q (validate:required) 未写入 openapi.yaml required", schemaName, f))
				}
			}
			for f := range specDef.required {
				if !goReq[f] {
					mismatches = append(mismatches, fmt.Sprintf("%s: openapi.yaml required 字段 %q 在 Go 结构体中未标记 validate:required", schemaName, f))
				}
			}
		}
	}
	slices.Sort(mismatches)
	if len(mismatches) > 0 {
		t.Errorf("[结构体 <-> OpenAPI Schema 字段漂移] 检测到 %d 处不一致：\n  - %s",
			len(mismatches), strings.Join(mismatches, "\n  - "))
	}
}

func TestPermissionCodesSyncedAcrossGoMigrationsAndOpenAPI(t *testing.T) {
	serverRoot := filepath.Clean("..")

	usedInGo, err := collectUsedPermissionCodes(filepath.Join(serverRoot, "internal", "modules"))
	if err != nil {
		t.Fatalf("scan Go permission codes: %v", err)
	}
	seededInSQL, err := collectSeededPermissionCodes(filepath.Join(serverRoot, "db", "migrations"))
	if err != nil {
		t.Fatalf("scan SQL migrations: %v", err)
	}
	specPerms, err := collectOpenAPIPermissionEnum(filepath.Join(serverRoot, "docs", "openapi.yaml"))
	if err != nil {
		t.Fatalf("scan openapi PermissionCode enum: %v", err)
	}

	for code, loc := range usedInGo {
		if !seededInSQL[code] {
			t.Errorf("[权限漂移] Go 路由使用了权限码 %q (%s)，但 db/migrations/*.sql 未插入该种子权限", code, loc)
		}
		if !specPerms[code] {
			t.Errorf("[权限漂移] Go 路由使用了权限码 %q (%s)，但 docs/openapi.yaml PermissionCode 枚举未包含该值", code, loc)
		}
	}
	for code := range seededInSQL {
		if !specPerms[code] {
			t.Errorf("[权限漂移] SQL 迁移定义了权限码 %q，但 docs/openapi.yaml PermissionCode 枚举缺失", code)
		}
	}
	for code := range specPerms {
		if !seededInSQL[code] {
			t.Errorf("[权限漂移] docs/openapi.yaml 声明了权限码 %q，但 db/migrations/*.sql 未找到对应记录", code)
		}
	}
}

func collectRegisteredHTTPRoutes(serverRoot string) (map[string]routeMeta, error) {
	routes := make(map[string]routeMeta)
	fset := token.NewFileSet()
	scanDirs := []string{
		filepath.Join(serverRoot, "cmd", "server"),
		filepath.Join(serverRoot, "internal", "modules"),
	}

	for _, root := range scanDirs {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return walkErr
			}
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(serverRoot, path)
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) < 2 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				var needsAuth bool
				switch sel.Sel.Name {
				case "Public", "HandleFunc":
					needsAuth = false
				case "Authenticated", "Protected":
					needsAuth = true
				default:
					return true
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				pattern, err := strconv.Unquote(lit.Value)
				if err != nil || !strings.Contains(pattern, " /") {
					return true
				}
				pos := fset.Position(lit.Pos())
				routes[pattern] = routeMeta{
					loc:       relPath + ":" + strconv.Itoa(pos.Line),
					needsAuth: needsAuth,
				}
				return true
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return routes, nil
}

func parseOpenAPIRoutes(specPath string) (map[string]bool, error) {
	f, err := os.Open(specPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	routes := make(map[string]bool)
	httpMethods := map[string]bool{
		"get": true, "post": true, "put": true, "patch": true, "delete": true,
	}

	scanner := bufio.NewScanner(f)
	inPaths := false
	currentPath := ""
	currentRoute := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			inPaths = trimmed == "paths:"
			currentPath = ""
			currentRoute = ""
			continue
		}
		if !inPaths {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			currentPath = strings.Trim(strings.TrimSuffix(trimmed, ":"), "'\"")
			currentRoute = ""
			continue
		}
		if indent == 4 && currentPath != "" && strings.HasSuffix(trimmed, ":") {
			method := strings.ToLower(strings.TrimSuffix(trimmed, ":"))
			if httpMethods[method] {
				currentRoute = strings.ToUpper(method) + " " + currentPath
				routes[currentRoute] = false
			} else {
				currentRoute = ""
			}
			continue
		}
		if indent == 6 && currentRoute != "" && strings.HasPrefix(trimmed, "security:") && strings.Contains(trimmed, "BearerAuth") {
			routes[currentRoute] = true
		}
	}
	return routes, scanner.Err()
}

type openAPISchemaDef struct {
	props    map[string]bool
	required map[string]bool
}

func parseOpenAPISchemas(specPath string) (map[string]openAPISchemaDef, error) {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	out := make(map[string]openAPISchemaDef)
	inComponents := false
	inSchemas := false
	currentSchema := ""
	section := ""

	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 {
			inComponents = trimmed == "components:"
			inSchemas = false
			currentSchema = ""
			continue
		}
		if !inComponents {
			continue
		}
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			inSchemas = trimmed == "schemas:"
			currentSchema = ""
			continue
		}
		if !inSchemas {
			continue
		}
		if indent == 4 && strings.HasSuffix(trimmed, ":") {
			currentSchema = strings.TrimSuffix(trimmed, ":")
			out[currentSchema] = openAPISchemaDef{
				props:    make(map[string]bool),
				required: make(map[string]bool),
			}
			section = ""
			continue
		}
		if currentSchema == "" {
			continue
		}
		if indent == 6 {
			if strings.HasPrefix(trimmed, "required:") {
				section = "required"
				rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "required:"))
				if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]") {
					inner := strings.TrimSuffix(strings.TrimPrefix(rest, "["), "]")
					for _, part := range strings.Split(inner, ",") {
						k := strings.TrimSpace(part)
						if k != "" {
							out[currentSchema].required[k] = true
						}
					}
				}
			} else if trimmed == "properties:" {
				section = "properties"
			} else if strings.HasSuffix(trimmed, ":") {
				section = ""
			}
			continue
		}
		if indent == 8 && section == "required" && strings.HasPrefix(trimmed, "- ") {
			out[currentSchema].required[strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))] = true
		}
		if indent == 8 && section == "properties" && strings.HasSuffix(trimmed, ":") {
			propName := strings.Trim(strings.TrimSuffix(trimmed, ":"), "'\"")
			out[currentSchema].props[propName] = true
		}
	}
	return out, nil
}

func inspectGoStructJSON(t reflect.Type) (props map[string]bool, req map[string]bool, hasValidate bool) {
	props = make(map[string]bool)
	req = make(map[string]bool)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		jtag := f.Tag.Get("json")
		if jtag == "-" {
			continue
		}
		name := strings.Split(jtag, ",")[0]
		if name == "" {
			name = f.Name
		}
		props[name] = true

		vtag := f.Tag.Get("validate")
		if vtag != "" {
			hasValidate = true
			for _, part := range strings.Split(vtag, ",") {
				if strings.TrimSpace(part) == "required" {
					req[name] = true
				}
			}
		}
	}
	return props, req, hasValidate
}

func collectUsedPermissionCodes(modulesDir string) (map[string]string, error) {
	codes := make(map[string]string)
	fset := token.NewFileSet()
	err := filepath.WalkDir(modulesDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			argIdx := -1
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				switch sel.Sel.Name {
				case "Protected":
					argIdx = 1
				case "RequirePermission":
					argIdx = 0
				}
			}
			if argIdx < 0 || len(call.Args) <= argIdx {
				return true
			}
			if lit, ok := call.Args[argIdx].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if val, err := strconv.Unquote(lit.Value); err == nil {
					pos := fset.Position(lit.Pos())
					codes[val] = filepath.Base(path) + ":" + strconv.Itoa(pos.Line)
				}
			}
			return true
		})
		return nil
	})
	return codes, err
}

func collectSeededPermissionCodes(migrationsDir string) (map[string]bool, error) {
	re := regexp.MustCompile(`'perm_[a-z0-9_]+'\s*,\s*'([a-z0-9]+:[a-z0-9:_]+)'`)
	codes := make(map[string]bool)
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".sql") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(migrationsDir, ent.Name()))
		if err != nil {
			return nil, err
		}
		for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
			codes[m[1]] = true
		}
	}
	return codes, nil
}

func collectOpenAPIPermissionEnum(specPath string) (map[string]bool, error) {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	codes := make(map[string]bool)
	lines := strings.Split(string(raw), "\n")
	inPermSchema := false
	inEnum := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 4 && strings.HasSuffix(trimmed, ":") {
			inPermSchema = trimmed == "PermissionCode:"
			inEnum = false
			continue
		}
		if !inPermSchema {
			continue
		}
		if indent == 6 && strings.HasSuffix(trimmed, ":") {
			inEnum = trimmed == "enum:"
			continue
		}
		if inEnum && indent == 8 && strings.HasPrefix(trimmed, "- ") {
			codes[strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))] = true
		}
	}
	return codes, nil
}
