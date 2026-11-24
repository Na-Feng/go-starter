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
	"slices"
	"strconv"
	"strings"
	"testing"
)

var (
	migrationNameRe    = regexp.MustCompile(`^(\d{5})_[a-z0-9_]+\.sql$`)
	createTableBlockRe = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-z_][a-z0-9_]*)\s*\((.*?)\);`)
	alterAddColRe      = regexp.MustCompile(`(?is)ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?([a-z_][a-z0-9_]*)\s+ADD\s+(?:COLUMN\s+)?(?:IF\s+NOT\s+EXISTS\s+)?([a-z_][a-z0-9_]*)\s+([^;]+);`)
	alterRenameRe      = regexp.MustCompile(`(?is)ALTER\s+TABLE\s+[^\n;]*\bRENAME\b`)
	insertStmtRe       = regexp.MustCompile(`(?is)\bINSERT\s+INTO\s+([a-z_][a-z0-9_]*)\s*\(([^)]+)\)\s*VALUES\s*\(([^)]+)\)(?:\s*RETURNING\s+([a-z0-9_,\s]+))?`)
	updateTableRe      = regexp.MustCompile(`(?is)\bUPDATE\s+([a-z_][a-z0-9_]*)\s+SET\s+(.+?)\s+WHERE\s+`)
	deleteTableRe      = regexp.MustCompile(`(?is)\bDELETE\s+FROM\s+([a-z_][a-z0-9_]*)\b`)
	simpleSelectRe     = regexp.MustCompile(`(?is)^\s*SELECT\s+(?:DISTINCT\s+)?(.+?)\s+FROM\s+([a-z_][a-z0-9_]*)(?:\s+([a-z_][a-z0-9_]*))?(?:\s+(.*))?$`)
	joinClauseRe       = regexp.MustCompile(`(?is)\bJOIN\s+([a-z_][a-z0-9_]*)\s+([a-z_][a-z0-9_]*)\b`)
	qualifiedColRe     = regexp.MustCompile(`\b([a-z_][a-z0-9_]*)\.([a-z_][a-z0-9_]*)\b`)
	sqlPlaceholderRe   = regexp.MustCompile(`\$\d+`)
)

func TestOfflineSQLMigrationsAndRepositoryQueriesMatchSchema(t *testing.T) {
	serverRoot := filepath.Clean("..")
	migDir := filepath.Join(serverRoot, "db", "migrations")

	schema, migErrs, err := loadAndValidateMigrations(migDir)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	if len(migErrs) > 0 {
		t.Fatalf("[SQL 迁移守卫] 检测到 %d 处问题：\n  - %s", len(migErrs), strings.Join(migErrs, "\n  - "))
	}

	var queryErrs []string
	queriesDir := filepath.Join(serverRoot, "db", "queries")
	if entries, err := os.ReadDir(queriesDir); err == nil {
		for _, ent := range entries {
			if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".sql") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(queriesDir, ent.Name()))
			if err != nil {
				t.Fatalf("read query file: %v", err)
			}
			for _, stmt := range strings.Split(string(raw), ";") {
				queryErrs = append(queryErrs, validateSQLStatement("db/queries/"+ent.Name(), stmt, schema)...)
			}
		}
	}

	scanGoDirs := []string{
		filepath.Join(serverRoot, "internal", "modules"),
		filepath.Join(serverRoot, "internal", "kernel", "audit"),
	}
	fset := token.NewFileSet()
	for _, dir := range scanGoDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return walkErr
			}
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(serverRoot, path)
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				val, err := strconv.Unquote(lit.Value)
				if err != nil || !looksLikeSQL(val) {
					return true
				}
				loc := fmt.Sprintf("%s:%d", filepath.ToSlash(rel), fset.Position(lit.Pos()).Line)
				queryErrs = append(queryErrs, validateSQLStatement(loc, val, schema)...)
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk Go SQL literals: %v", err)
		}
	}

	if len(queryErrs) > 0 {
		t.Errorf("[离线 SQL 列名/表名静态守卫] 检测到 %d 处与 db/migrations 不一致的 SQL：\n  - %s",
			len(queryErrs), strings.Join(queryErrs, "\n  - "))
	}
}

func TestSQLHygieneCatchesTyposAndUnsafeDDL(t *testing.T) {
	schema := map[string]map[string]bool{
		"users": {"id": true, "username": true, "status": true},
	}

	if errs := validateMigrationUpSQL("00004_bad.sql", "ALTER TABLE users ADD COLUMN phone TEXT NOT NULL;"); len(errs) == 0 {
		t.Fatal("expected unsafe ADD COLUMN NOT NULL without DEFAULT to be rejected")
	}
	if errs := validateMigrationUpSQL("00004_ok.sql", "ALTER TABLE users ADD COLUMN phone TEXT NOT NULL DEFAULT '';"); len(errs) != 0 {
		t.Fatalf("expected ADD COLUMN NOT NULL DEFAULT '' to pass, got %v", errs)
	}
	if errs := validateSQLStatement("repo.go:10", "SELECT id, nick_name FROM users WHERE id = $1", schema); len(errs) == 0 {
		t.Fatal("expected unknown column nick_name to be rejected")
	}
	if errs := validateSQLStatement("repo.go:20", "INSERT INTO users (id, username) VALUES ($1)", schema); len(errs) == 0 {
		t.Fatal("expected INSERT column/value count mismatch to be rejected")
	}
	if errs := validateSQLStatement("repo.go:30", "SELECT id FROM ghost_table", schema); len(errs) == 0 {
		t.Fatal("expected unknown table ghost_table to be rejected")
	}
}

func loadAndValidateMigrations(migDir string) (map[string]map[string]bool, []string, error) {
	entries, err := os.ReadDir(migDir)
	if err != nil {
		return nil, nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)

	schema := make(map[string]map[string]bool)
	seenVer := make(map[string]string)
	var errs []string

	for _, name := range names {
		m := migrationNameRe.FindStringSubmatch(name)
		if m == nil {
			errs = append(errs, fmt.Sprintf("%s: 迁移文件名必须符合 NNNNN_snake_case.sql", name))
			continue
		}
		if prev, dup := seenVer[m[1]]; dup {
			errs = append(errs, fmt.Sprintf("%s 与 %s 迁移序号 %s 冲突", name, prev, m[1]))
		}
		seenVer[m[1]] = name

		rawBytes, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			return nil, nil, err
		}
		raw := string(rawBytes)
		upIdx := strings.Index(raw, "-- +goose Up")
		downIdx := strings.Index(raw, "-- +goose Down")
		if upIdx < 0 || downIdx < 0 || downIdx <= upIdx {
			errs = append(errs, fmt.Sprintf("%s: 必须同时包含有序的 '-- +goose Up' 与 '-- +goose Down' 回滚段", name))
			continue
		}
		upSQL := raw[upIdx:downIdx]
		errs = append(errs, validateMigrationUpSQL(name, upSQL)...)
		extractSchemaFromUpSQL(upSQL, schema)
	}
	return schema, errs, nil
}

func validateMigrationUpSQL(fileName, upSQL string) []string {
	var errs []string
	if alterRenameRe.MatchString(upSQL) {
		errs = append(errs, fmt.Sprintf("%s: 禁止在滚动发布迁移中使用 ALTER TABLE RENAME（会导致新旧实例不兼容）", fileName))
	}
	for _, m := range alterAddColRe.FindAllStringSubmatch(upSQL, -1) {
		colDef := strings.ToUpper(m[3])
		if strings.Contains(colDef, "NOT NULL") && !strings.Contains(colDef, "DEFAULT") {
			errs = append(errs, fmt.Sprintf(
				"%s: ALTER TABLE %s ADD COLUMN %s 声明了 NOT NULL 但缺少 DEFAULT，会导致非空表锁死或旧实例写入失败",
				fileName, m[1], m[2],
			))
		}
	}
	return errs
}

func extractSchemaFromUpSQL(upSQL string, schema map[string]map[string]bool) {
	sqlKeywords := map[string]bool{
		"primary": true, "foreign": true, "unique": true, "constraint": true, "check": true,
	}
	for _, m := range createTableBlockRe.FindAllStringSubmatch(upSQL, -1) {
		tbl := strings.ToLower(m[1])
		if schema[tbl] == nil {
			schema[tbl] = make(map[string]bool)
		}
		for _, line := range strings.Split(m[2], "\n") {
			trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ","))
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				continue
			}
			fields := strings.Fields(trimmed)
			if len(fields) < 2 {
				continue
			}
			col := strings.ToLower(strings.Trim(fields[0], `"`))
			if !sqlKeywords[col] {
				schema[tbl][col] = true
			}
		}
	}
	for _, m := range alterAddColRe.FindAllStringSubmatch(upSQL, -1) {
		tbl := strings.ToLower(m[1])
		if schema[tbl] == nil {
			schema[tbl] = make(map[string]bool)
		}
		schema[tbl][strings.ToLower(m[2])] = true
	}
}

func looksLikeSQL(s string) bool {
	u := strings.ToUpper(strings.TrimSpace(s))
	for _, kw := range []string{"SELECT ", "INSERT INTO ", "UPDATE ", "DELETE FROM "} {
		if strings.HasPrefix(u, kw) {
			return true
		}
	}
	return false
}

func validateSQLStatement(loc, rawStmt string, schema map[string]map[string]bool) []string {
	var cleanLines []string
	for _, line := range strings.Split(rawStmt, "\n") {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		if strings.TrimSpace(line) != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	stmt := strings.TrimSpace(strings.TrimSuffix(strings.Join(cleanLines, " "), ";"))
	if stmt == "" {
		return nil
	}

	var errs []string
	if m := insertStmtRe.FindStringSubmatch(stmt); m != nil {
		tbl := strings.ToLower(m[1])
		colsMap, ok := schema[tbl]
		if !ok {
			return []string{fmt.Sprintf("%s: INSERT 引用了不存在的表 %q", loc, tbl)}
		}
		cols := splitSQLIdentList(m[2])
		vals := strings.Split(m[3], ",")
		if len(cols) != len(vals) {
			errs = append(errs, fmt.Sprintf("%s: INSERT INTO %s 列数量 (%d) 与 VALUES 参数数量 (%d) 不匹配", loc, tbl, len(cols), len(vals)))
		}
		for _, c := range cols {
			if !colsMap[c] {
				errs = append(errs, fmt.Sprintf("%s: 表 %q 不存在列 %q", loc, tbl, c))
			}
		}
		if m[4] != "" {
			for _, rc := range splitSQLIdentList(m[4]) {
				if !colsMap[rc] {
					errs = append(errs, fmt.Sprintf("%s: INSERT RETURNING 引用了表 %q 不存在的列 %q", loc, tbl, rc))
				}
			}
		}
		return errs
	}

	if m := updateTableRe.FindStringSubmatch(stmt); m != nil {
		tbl := strings.ToLower(m[1])
		colsMap, ok := schema[tbl]
		if !ok {
			return []string{fmt.Sprintf("%s: UPDATE 引用了不存在的表 %q", loc, tbl)}
		}
		for _, assign := range strings.Split(m[2], ",") {
			lhs := strings.ToLower(strings.TrimSpace(strings.SplitN(assign, "=", 2)[0]))
			if lhs != "" && !colsMap[lhs] {
				errs = append(errs, fmt.Sprintf("%s: UPDATE %s 设置了不存在的列 %q", loc, tbl, lhs))
			}
		}
		return errs
	}

	if m := deleteTableRe.FindStringSubmatch(stmt); m != nil {
		tbl := strings.ToLower(m[1])
		if _, ok := schema[tbl]; !ok {
			errs = append(errs, fmt.Sprintf("%s: DELETE 引用了不存在的表 %q", loc, tbl))
		}
		return errs
	}

	if m := simpleSelectRe.FindStringSubmatch(stmt); m != nil {
		selectList, baseTbl, baseAlias, tail := m[1], strings.ToLower(m[2]), strings.ToLower(m[3]), m[4]
		if _, ok := schema[baseTbl]; !ok {
			return []string{fmt.Sprintf("%s: SELECT 引用了不存在的表 %q", loc, baseTbl)}
		}
		sqlReserved := map[string]bool{"where": true, "order": true, "limit": true, "join": true, "left": true, "inner": true, "group": true}
		aliases := map[string]string{}
		if baseAlias != "" && !sqlReserved[baseAlias] {
			aliases[baseAlias] = baseTbl
		}
		hasJoin := false
		for _, jm := range joinClauseRe.FindAllStringSubmatch(stmt, -1) {
			hasJoin = true
			jTbl, jAlias := strings.ToLower(jm[1]), strings.ToLower(jm[2])
			if _, ok := schema[jTbl]; !ok {
				errs = append(errs, fmt.Sprintf("%s: JOIN 引用了不存在的表 %q", loc, jTbl))
				continue
			}
			if jAlias != "" && !sqlReserved[jAlias] {
				aliases[jAlias] = jTbl
			}
		}
		if hasJoin || len(aliases) > 0 {
			for _, qm := range qualifiedColRe.FindAllStringSubmatch(selectList+" "+tail, -1) {
				al, col := strings.ToLower(qm[1]), strings.ToLower(qm[2])
				if targetTbl, ok := aliases[al]; ok && !schema[targetTbl][col] {
					errs = append(errs, fmt.Sprintf("%s: 表 %q (%s) 不存在列 %q", loc, targetTbl, al, col))
				}
			}
			return errs
		}
		upperSel := strings.ToUpper(strings.TrimSpace(selectList))
		if strings.HasPrefix(upperSel, "COUNT(") || upperSel == "*" {
			return errs
		}
		for _, col := range splitSQLIdentList(selectList) {
			if col != "" && !sqlPlaceholderRe.MatchString(col) && !schema[baseTbl][col] {
				errs = append(errs, fmt.Sprintf("%s: SELECT 引用了表 %q 不存在的列 %q", loc, baseTbl, col))
			}
		}
	}
	return errs
}

func splitSQLIdentList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		p := strings.ToLower(strings.TrimSpace(part))
		if fields := strings.Fields(p); len(fields) > 0 {
			out = append(out, strings.Trim(fields[0], `"`))
		}
	}
	return out
}
