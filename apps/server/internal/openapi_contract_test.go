package internal_test

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestOpenAPIContractMatchesRegisteredRoutes(t *testing.T) {
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

	var missingInSpec []string
	for route, loc := range registered {
		if !documented[route] {
			missingInSpec = append(missingInSpec, route+" (registered at "+loc+")")
		}
	}
	slices.Sort(missingInSpec)
	if len(missingInSpec) > 0 {
		t.Errorf(
			"[契约漂移] 以下 Go 路由尚未写入 docs/openapi.yaml（请先更新 docs/openapi.yaml 并执行 make gen-api）：\n  - %s",
			strings.Join(missingInSpec, "\n  - "),
		)
	}

	var phantomInSpec []string
	for route := range documented {
		if _, ok := registered[route]; !ok {
			phantomInSpec = append(phantomInSpec, route)
		}
	}
	slices.Sort(phantomInSpec)
	if len(phantomInSpec) > 0 {
		t.Errorf(
			"[契约漂移] docs/openapi.yaml 声明了以下未在 Go 代码中注册的幽灵路由：\n  - %s",
			strings.Join(phantomInSpec, "\n  - "),
		)
	}
}

func collectRegisteredHTTPRoutes(serverRoot string) (map[string]string, error) {
	routes := make(map[string]string)
	fset := token.NewFileSet()
	scanDirs := []string{
		filepath.Join(serverRoot, "cmd", "server"),
		filepath.Join(serverRoot, "internal", "modules"),
	}

	for _, root := range scanDirs {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
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
				if !ok || (sel.Sel.Name != "Handle" && sel.Sel.Name != "HandleFunc") {
					return true
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				pattern, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				if strings.Contains(pattern, " /") {
					pos := fset.Position(lit.Pos())
					routes[pattern] = relPath + ":" + strconv.Itoa(pos.Line)
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

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			inPaths = trimmed == "paths:"
			currentPath = ""
			continue
		}
		if !inPaths {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			currentPath = strings.Trim(strings.TrimSuffix(trimmed, ":"), "'\"")
			continue
		}
		if indent == 4 && currentPath != "" && strings.HasSuffix(trimmed, ":") {
			method := strings.ToLower(strings.TrimSuffix(trimmed, ":"))
			if httpMethods[method] {
				routes[strings.ToUpper(method)+" "+currentPath] = true
			}
		}
	}
	return routes, scanner.Err()
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
			if !ok || len(call.Args) == 0 {
				return true
			}
			isPermCall := false
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				isPermCall = fun.Name == "perm"
			case *ast.SelectorExpr:
				isPermCall = fun.Sel.Name == "RequirePermission"
			}
			if !isPermCall {
				return true
			}
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
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
