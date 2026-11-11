package codeintel

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// v3.22.0 codegraph v1.5：Python/JS/TS 词法符号表 + Go 词法调用边（get_callers）。
func TestScanPolyFiles(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "util.py"), "def calc(a, b):\n    \"\"\"求和\"\"\"\n    return a + b\n\ndef _hidden():\n    pass\n")
	mustWrite(t, filepath.Join(dir, "app.js"), "function handler(req) { return req; }\nconst run = () => 1;\n")
	mustWrite(t, filepath.Join(dir, "node_modules", "skip.js"), "function nope() {}\n")
	idx, err := NewIndex(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	defs := idx.PolyDefinitions("calc")
	if len(defs) != 1 || defs[0].Lang != "py" || defs[0].File != "util.py" || defs[0].Line != 1 {
		t.Fatalf("py 定义错误: %+v", defs)
	}
	if !strings.Contains(defs[0].Args, "a, b") {
		t.Fatalf("参数表错误: %q", defs[0].Args)
	}
	if len(idx.PolyDefinitions("handler")) != 1 || idx.PolyDefinitions("handler")[0].Lang != "js" {
		t.Fatalf("js 定义错误: %+v", idx.PolyDefinitions("handler"))
	}
	if len(idx.PolyDefinitions("run")) != 1 {
		t.Fatalf("const 箭头函数应命中: %+v", idx.PolyDefinitions("run"))
	}
	if len(idx.PolyDefinitions("nope")) != 0 {
		t.Fatal("node_modules 不应索引")
	}
	if idx.PolyDefinitions("NoSuch") != nil {
		t.Fatal("未知符号应为空")
	}
}

func TestGoCallers(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "package app\n\nfunc Serve() error { return nil }\n")
	mustWrite(t, filepath.Join(dir, "main.go"), "package main\n\nimport \"app\"\n\nfunc main() {\n\t_ = app.Serve()\n\t_ = Serve()\n\t// Serve() 注释里的调用\n\t_ = \"Serve() 字符串\"\n}\n")
	idx, err := NewIndex(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.GoCallers("Serve", 50)
	found := map[string]bool{}
	for _, r := range refs {
		found[r.File+":"+strconv.Itoa(r.Line)] = true
	}
	if !found["main.go:6"] || !found["main.go:7"] {
		t.Fatalf("调用形态应命中 pkg.F( 与 F(: %v", refs)
	}
	// 定义行不计；注释/字符串行按词法口径如实计入（工具描述声明须核实）
	if found["a.go:3"] {
		t.Fatalf("定义行不应计入: %v", refs)
	}
	if len(idx.GoCallers("Serve", 1)) != 1 {
		t.Fatal("limit 未生效")
	}
}

// textx#83：shell 词法函数表 + .mjs/.cjs/.pyw 扩展名归族。
func TestScanPolyFilesShellAndExts(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "deploy.sh"), "#!/usr/bin/env bash\nrollback() {\n  echo rollback\n}\n\nfunction cleanup {\n  echo cleanup\n}\n\nfunction teardown() {\n  echo teardown\n}\n")
	mustWrite(t, filepath.Join(dir, "boot.mjs"), "export const boot = async () => 1;\nfunction init() {}\n")
	mustWrite(t, filepath.Join(dir, "legacy.cjs"), "function start() {}\n")
	mustWrite(t, filepath.Join(dir, "gui.pyw"), "def gui_main():\n    pass\n")
	idx, err := NewIndex(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	// POSIX `name() {`——Line 从名字组计数（shebang 在第 1 行）。
	if d := idx.PolyDefinitions("rollback"); len(d) != 1 || d[0].Lang != "shell" || d[0].Line != 2 || d[0].File != "deploy.sh" {
		t.Fatalf("POSIX 形态错误: %+v", d)
	}
	// `function name {` 与混合形态 `function name() {`。
	for _, name := range []string{"cleanup", "teardown"} {
		if d := idx.PolyDefinitions(name); len(d) != 1 || d[0].Lang != "shell" {
			t.Fatalf("%s 形态错误: %+v", name, d)
		}
	}
	// shell 无参数表/文档（词法最小面）。
	if rb := idx.PolyDefinitions("rollback")[0]; rb.Args != "" || rb.Doc != "" {
		t.Fatalf("shell 符号应为最小面: %+v", rb)
	}
	// 扩展名归族：.mjs/.cjs → js，.pyw → py。
	if d := idx.PolyDefinitions("init"); len(d) != 1 || d[0].Lang != "js" || d[0].File != "boot.mjs" {
		t.Fatalf(".mjs 应归 js: %+v", d)
	}
	if d := idx.PolyDefinitions("start"); len(d) != 1 || d[0].File != "legacy.cjs" {
		t.Fatalf(".cjs 应归 js: %+v", d)
	}
	if d := idx.PolyDefinitions("gui_main"); len(d) != 1 || d[0].Lang != "py" || d[0].File != "gui.pyw" {
		t.Fatalf(".pyw 应归 py: %+v", d)
	}
}
