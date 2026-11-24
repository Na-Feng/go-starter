package internal_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const moduleImportPrefix = "go-starter/internal/modules/"

var debtMarkerRe = regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|WORKAROUND|XXX)\b`)

func TestArchitectureAndAntiSlopGuardrails(t *testing.T) {
	internalRoot := "."
	var allViolations []string

	err := filepath.WalkDir(internalRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		norm := filepath.ToSlash(path)
		if !strings.Contains(norm, "/") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		violations := analyzeGoAST("internal/"+norm, fset, file)
		allViolations = append(allViolations, violations...)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to scan internal directory: %v", err)
	}

	if len(allViolations) > 0 {
		t.Errorf("Go 架构与防凑合守卫拦截到 %d 处违规：\n  - %s",
			len(allViolations), strings.Join(allViolations, "\n  - "))
	}
}

func TestArchGuardrailsCatchViolations(t *testing.T) {
	cases := []struct {
		name        string
		relPath     string
		src         string
		wantSubstr  string
		expectClean bool
	}{
		{
			name:       "cross_module_import_blocked",
			relPath:    "internal/modules/sample/service.go",
			src:        `package sample; import _ "go-starter/internal/modules/iam"`,
			wantSubstr: "禁止跨业务模块直接引用",
		},
		{
			name:       "new_future_module_auto_isolated",
			relPath:    "internal/modules/orders/service.go",
			src:        `package orders; import _ "go-starter/internal/modules/sample"`,
			wantSubstr: "禁止跨业务模块直接引用",
		},
		{
			name:       "kernel_importing_module_blocked",
			relPath:    "internal/kernel/httpx/bad.go",
			src:        `package httpx; import _ "go-starter/internal/modules/iam"`,
			wantSubstr: "internal/kernel 禁止反向依赖业务模块",
		},
		{
			name:       "stdlib_log_and_ioutil_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        `package sample; import (_ "log"; _ "io/ioutil")`,
			wantSubstr: "禁止使用标准库 log",
		},
		{
			name:       "time_sleep_patch_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        `package sample; import "time"; func f() { time.Sleep(time.Second) }`,
			wantSubstr: "禁止使用 time.Sleep 硬等",
		},
		{
			name:       "panic_and_fmt_print_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        `package sample; import "fmt"; func f() { fmt.Println("x"); panic("boom") }`,
			wantSubstr: "禁止在业务代码调用 panic",
		},
		{
			name:       "root_context_in_module_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        `package sample; import "context"; func f() { _ = context.Background() }`,
			wantSubstr: "禁止在业务层新建根 Context",
		},
		{
			name:       "default_http_get_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        `package sample; import "net/http"; func f() { _, _ = http.Get("https://example.com") }`,
			wantSubstr: "禁止使用无超时的默认 http.Get/Post",
		},
		{
			name:       "bare_goroutine_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        `package sample; func f() { go func() {}() }`,
			wantSubstr: "禁止在业务模块直接使用裸 go 协程",
		},
		{
			name:       "bare_servemux_handle_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        `package sample; import "net/http"; func f(mux *http.ServeMux, h http.HandlerFunc) { mux.HandleFunc("GET /x", h) }`,
			wantSubstr: "禁止在业务模块直接注册裸 ServeMux 路由",
		},
		{
			name:       "nolint_bypass_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        "package sample\n//nolint:errcheck\nfunc f() {}\n",
			wantSubstr: "禁止使用 //nolint 豁免静态检查",
		},
		{
			name:       "hack_todo_comment_blocked",
			relPath:    "internal/modules/sample/bad.go",
			src:        "package sample\nfunc f() {\n\t// HACK: temporary workaround\n}\n",
			wantSubstr: "禁止遗留 TODO/FIXME/HACK/WORKAROUND",
		},
		{
			name:    "long_excuse_comment_in_func_blocked",
			relPath: "internal/modules/sample/bad.go",
			src: "package sample\nfunc f() {\n" +
				"\t// line 1 explaining workaround\n" +
				"\t// line 2 explaining workaround\n" +
				"\t// line 3 explaining workaround\n" +
				"}\n",
			wantSubstr: "禁止超过 2 行的长篇解释性注释",
		},
		{
			name:    "concise_two_line_intent_and_doc_comment_allowed",
			relPath: "internal/modules/sample/good.go",
			src: "package sample\n" +
				"// ExportedService 处理示例聚合。\n" +
				"// 支持多租户扩展。\n" +
				"// 遵循契约驱动。\n" +
				"func ExportedService() {\n" +
				"\t// 幂等前置校验\n" +
				"\t// 确保状态合法\n" +
				"}\n",
			expectClean: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, tc.relPath, tc.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse synthetic source failed: %v", err)
			}
			got := analyzeGoAST(tc.relPath, fset, file)
			if tc.expectClean {
				if len(got) > 0 {
					t.Fatalf("expected 0 violations, got: %v", got)
				}
				return
			}
			joined := strings.Join(got, "\n")
			if !strings.Contains(joined, tc.wantSubstr) {
				t.Fatalf("expected violation containing %q, got: %v", tc.wantSubstr, got)
			}
		})
	}
}

func analyzeGoAST(relPath string, fset *token.FileSet, file *ast.File) []string {
	var out []string
	isTest := strings.HasSuffix(relPath, "_test.go")
	isKernel := strings.HasPrefix(relPath, "internal/kernel/")
	modName := extractModuleName(relPath)
	isBusinessModule := modName != ""

	posStr := func(p token.Pos) string {
		pos := fset.Position(p)
		return fmt.Sprintf("%s:%d", relPath, pos.Line)
	}

	for _, imp := range file.Imports {
		pkg, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if pkg == "io/ioutil" {
			out = append(out, fmt.Sprintf("%s: 已废弃 io/ioutil，请使用标准库 io 或 os 替代", posStr(imp.Pos())))
		}
		if pkg == "log" {
			out = append(out, fmt.Sprintf("%s: 禁止使用标准库 log，请统一通过 internal/kernel/logger 使用 log/slog", posStr(imp.Pos())))
		}
		if isKernel && strings.HasPrefix(pkg, moduleImportPrefix) {
			out = append(out, fmt.Sprintf("%s: internal/kernel 禁止反向依赖业务模块 %q", posStr(imp.Pos()), pkg))
		}
		if isBusinessModule && strings.HasPrefix(pkg, moduleImportPrefix) {
			rest := strings.TrimPrefix(pkg, moduleImportPrefix)
			targetMod := strings.SplitN(rest, "/", 2)[0]
			if targetMod != modName {
				out = append(out, fmt.Sprintf(
					"%s: 禁止跨业务模块直接引用 (%s -> %s)，请在调用方定义最小接口并由 cmd/server/main.go 注入",
					posStr(imp.Pos()), modName, targetMod,
				))
			}
		}
	}

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(c.Text, "//"), "/*"))
			if strings.HasPrefix(strings.ToLower(trimmed), "nolint") {
				out = append(out, fmt.Sprintf("%s: 禁止使用 //nolint 豁免静态检查，请直接修复根因", posStr(c.Pos())))
			}
			if isBusinessModule && !isTest && debtMarkerRe.MatchString(c.Text) {
				out = append(out, fmt.Sprintf(
					"%s: 禁止遗留 TODO/FIXME/HACK/WORKAROUND 补丁注释，请直接完成实现或修复根因",
					posStr(c.Pos()),
				))
			}
		}
	}

	if !isBusinessModule || isTest {
		return out
	}

	var funcRanges [][2]token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body != nil {
				funcRanges = append(funcRanges, [2]token.Pos{fn.Body.Lbrace, fn.Body.Rbrace})
			}
		case *ast.FuncLit:
			if fn.Body != nil {
				funcRanges = append(funcRanges, [2]token.Pos{fn.Body.Lbrace, fn.Body.Rbrace})
			}
		case *ast.GoStmt:
			out = append(out, fmt.Sprintf(
				"%s: 禁止在业务模块直接使用裸 go 协程（未捕获 panic 且受请求 Context 取消影响），请改用 internal/kernel/async.Go 或 worker 异步队列",
				posStr(fn.Pos()),
			))
		case *ast.CallExpr:
			switch fun := fn.Fun.(type) {
			case *ast.Ident:
				if fun.Name == "panic" {
					out = append(out, fmt.Sprintf("%s: 禁止在业务代码调用 panic，请返回 *errs.AppError", posStr(fun.Pos())))
				}
			case *ast.SelectorExpr:
				if fun.Sel.Name == "Handle" || fun.Sel.Name == "HandleFunc" {
					out = append(out, fmt.Sprintf(
						"%s: 禁止在业务模块直接注册裸 ServeMux 路由 (%s)，请统一使用 *auth.Router 的 Public / Authenticated / Protected 显式声明鉴权等级",
						posStr(fun.Pos()), fun.Sel.Name,
					))
					return true
				}
				pkgIdent, ok := fun.X.(*ast.Ident)
				if !ok {
					return true
				}
				callName := pkgIdent.Name + "." + fun.Sel.Name
				switch {
				case pkgIdent.Name == "fmt" && strings.HasPrefix(fun.Sel.Name, "Print"):
					out = append(out, fmt.Sprintf("%s: 禁止使用 %s，请使用 slog 结构化日志", posStr(fun.Pos()), callName))
				case pkgIdent.Name == "http" && (fun.Sel.Name == "Get" || fun.Sel.Name == "Post" || fun.Sel.Name == "Head" || fun.Sel.Name == "PostForm"):
					out = append(out, fmt.Sprintf("%s: 禁止使用无超时的默认 http.Get/Post (%s)，请显式构造带 Timeout 的 http.Client", posStr(fun.Pos()), callName))
				case callName == "os.Exit":
					out = append(out, fmt.Sprintf("%s: 仅允许在 cmd/ 入口调用 os.Exit，业务层请返回 *errs.AppError", posStr(fun.Pos())))
				case callName == "time.Sleep":
					out = append(out, fmt.Sprintf("%s: 禁止使用 time.Sleep 硬等，请改用 select <-ctx.Done() 或异步任务队列", posStr(fun.Pos())))
				case callName == "context.Background" || callName == "context.TODO":
					out = append(out, fmt.Sprintf("%s: 禁止在业务层新建根 Context (%s)，请透传上游传入的 ctx", posStr(fun.Pos()), callName))
				}
			}
		}
		return true
	})

	inFuncBody := func(p token.Pos) bool {
		for _, r := range funcRanges {
			if p > r[0] && p < r[1] {
				return true
			}
		}
		return false
	}

	for _, cg := range file.Comments {
		if !inFuncBody(cg.Pos()) {
			continue
		}
		startLine := fset.Position(cg.Pos()).Line
		endLine := fset.Position(cg.End()).Line
		span := endLine - startLine + 1
		if span > 2 {
			out = append(out, fmt.Sprintf(
				"%s: 业务函数内禁止超过 2 行的长篇解释性注释（当前连续 %d 行），请提取语义化子函数或直接修复架构冲突",
				posStr(cg.Pos()), span,
			))
		}
	}

	return out
}

func extractModuleName(relPath string) string {
	const prefix = "internal/modules/"
	if !strings.HasPrefix(relPath, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(relPath, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

var (
	allowedEntryDocRe = regexp.MustCompile(`^(apps/[a-z0-9-]+/)?(README|AGENTS|CLAUDE)\.md$`)
	adrDocRe          = regexp.MustCompile(`^docs/adr/\d{4}-[a-z0-9-]+\.md$`)
	specDocRe         = regexp.MustCompile(`^docs/specs/\d{4}-\d{2}-\d{2}-[a-z0-9-]+\.md$`)
	postmortemDocRe   = regexp.MustCompile(`^docs/postmortems/\d{4}-\d{2}-\d{2}-[a-z0-9-]+\.md$`)
	runbookDocRe      = regexp.MustCompile(`^docs/runbooks/[a-z][a-z0-9-]*\.md$`)
	agentSkillDocRe   = regexp.MustCompile(`^\.agents/skills/([a-z0-9-]+)/SKILL\.md$`)
	frontmatterStatRe = regexp.MustCompile(`(?m)^status:\s*(draft|accepted|superseded)\s*$`)
	skillNameRe       = regexp.MustCompile(`(?m)^name:\s*([a-z0-9-]+)\s*$`)
	skillDescRe       = regexp.MustCompile(`(?m)^description:\s*\S+`)
)

func TestMarkdownDocsLayoutAndFrontmatterGuardrail(t *testing.T) {
	repoRoot := filepath.Clean("../../..")
	skipDirs := map[string]bool{
		"node_modules": true,
		"dist":         true,
		"bin":          true,
		"uploads":      true,
		"vendor":       true,
	}

	var violations []string
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			if name != "." && name != ".." && name != ".agents" && (strings.HasPrefix(name, ".") || skipDirs[name]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".md") {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if msg := validateMarkdownDoc(filepath.ToSlash(rel), string(raw)); msg != "" {
			violations = append(violations, msg)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan repo markdown docs: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("全仓 Markdown 文档治理守卫拦截到 %d 处违规：\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

func TestMarkdownGuardrailCatchesViolations(t *testing.T) {
	validFM := "---\nstatus: accepted\nscope: fullstack\n---\n# Title\n"
	cases := []struct {
		relPath     string
		content     string
		expectClean bool
		wantSubstr  string
	}{
		{relPath: "AGENTS.md", content: "# Root", expectClean: true},
		{relPath: "apps/web/AGENTS.md", content: "# Web", expectClean: true},
		{relPath: "docs/adr/0001-harness-hardening.md", content: validFM, expectClean: true},
		{relPath: "docs/specs/2026-11-19-order-flow.md", content: validFM, expectClean: true},
		{relPath: "docs/postmortems/2026-11-19-cache-bug.md", content: "# Postmortem", expectClean: true},
		{relPath: "docs/runbooks/deploy-prod.md", content: "# Runbook", expectClean: true},
		{
			relPath:     ".agents/skills/add-feature/SKILL.md",
			content:     "---\nname: add-feature\ndescription: Add feature playbook\n---\n# Playbook\n",
			expectClean: true,
		},
		{
			relPath:    ".agents/skills/add-feature/SKILL.md",
			content:    "---\nname: wrong-name\ndescription: mismatch\n---\n# Playbook\n",
			wantSubstr: "Frontmatter 的 name 必须与目录名",
		},
		{
			relPath:    ".agents/random.md",
			content:    "# Stray",
			wantSubstr: "禁止新建未归口 .md 文件",
		},
		{
			relPath:    "docs/ai/2026-11-19-note.md",
			content:    validFM,
			wantSubstr: "禁止新建未归口 .md 文件",
		},
		{
			relPath:    "apps/server/SUMMARY.md",
			content:    "# Summary",
			wantSubstr: "禁止新建未归口 .md 文件",
		},
		{
			relPath:    "docs/adr/0002-missing-status.md",
			content:    "# No Frontmatter",
			wantSubstr: "必须以 YAML Frontmatter 开头",
		},
		{
			relPath:    "docs/runbooks/2026-11-19-dated-runbook.md",
			content:    "# Runbook",
			wantSubstr: "禁止新建未归口 .md 文件",
		},
	}

	for _, tc := range cases {
		msg := validateMarkdownDoc(tc.relPath, tc.content)
		if tc.expectClean && msg != "" {
			t.Fatalf("%s expected clean, got: %s", tc.relPath, msg)
		}
		if !tc.expectClean && !strings.Contains(msg, tc.wantSubstr) {
			t.Fatalf("%s expected %q, got: %q", tc.relPath, tc.wantSubstr, msg)
		}
	}
}

func validateMarkdownDoc(relPath, content string) string {
	if allowedEntryDocRe.MatchString(relPath) || postmortemDocRe.MatchString(relPath) || runbookDocRe.MatchString(relPath) {
		return ""
	}
	if m := agentSkillDocRe.FindStringSubmatch(relPath); m != nil {
		dirName := m[1]
		fm, ok := extractYAMLFrontmatter(content)
		if !ok {
			return fmt.Sprintf("%s: 通用 Skill 必须包含标准 YAML Frontmatter（name 与 description）", relPath)
		}
		nm := skillNameRe.FindStringSubmatch(fm)
		if nm == nil || nm[1] != dirName {
			return fmt.Sprintf("%s: Frontmatter 的 name 必须与目录名 %q 一致", relPath, dirName)
		}
		if !skillDescRe.MatchString(fm) {
			return fmt.Sprintf("%s: Frontmatter 缺少非空 description 字段", relPath)
		}
		return ""
	}
	if adrDocRe.MatchString(relPath) || specDocRe.MatchString(relPath) {
		fm, ok := extractYAMLFrontmatter(content)
		if !ok {
			return fmt.Sprintf(
				"%s: specs/ 与 adr/ 文档必须以 YAML Frontmatter 开头并声明 status: draft|accepted|superseded",
				relPath,
			)
		}
		if !frontmatterStatRe.MatchString(fm) {
			return fmt.Sprintf(
				"%s: Frontmatter 缺少合法的 status 字段（可选值：draft | accepted | superseded）",
				relPath,
			)
		}
		return ""
	}
	return fmt.Sprintf(
		"%s: 禁止新建未归口 .md 文件；临时草稿请放 .tmp/，正式文档请按用途放入 docs/specs/YYYY-MM-DD-<slug>.md、docs/adr/NNNN-<slug>.md、docs/postmortems/YYYY-MM-DD-<slug>.md 或 docs/runbooks/<slug>.md",
		relPath,
	)
}

func extractYAMLFrontmatter(content string) (string, bool) {
	trimmed := strings.TrimLeft(content, "\ufeff")
	if !strings.HasPrefix(trimmed, "---\n") {
		return "", false
	}
	endIdx := strings.Index(trimmed[4:], "\n---")
	if endIdx < 0 {
		return "", false
	}
	return trimmed[4 : 4+endIdx], true
}
