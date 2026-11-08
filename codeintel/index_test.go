package codeintel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture 跨文件 Go 工程：pkg 定义 + main 调用 + vendor 噪声 + 坏语法文件。
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "api.go"), `package app

import "fmt"

// Handler 处理入口。
type Handler struct{ Name string }

func (h *Handler) Serve(w Writer) error {
	if w == nil {
		return ErrNil
	}
	fmt.Println(h.Name)
	return nil
}

var ErrNil = errNil{}

type errNil struct{}
`)
	mustWrite(t, filepath.Join(dir, "writer.go"), `package app

// Writer 输出接口。
type Writer interface{ Write(p []byte) (int, error) }
`)
	mustWrite(t, filepath.Join(dir, "main.go"), `package main

import "app"

func main() {
	h := &app.Handler{Name: "x"}
	_ = h.Serve(nil)
}
`)
	mustWrite(t, filepath.Join(dir, "util.go"), `package app
func broken( {
`)
	mustWrite(t, filepath.Join(dir, "vendor", "x.go"), `package vendored
func Helper() {}
`)
	return dir
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNewIndexDefinitions(t *testing.T) {
	idx, err := NewIndex(fixture(t), 100)
	if err != nil {
		t.Fatal(err)
	}
	defs := idx.Definitions("Serve")
	if len(defs) != 1 {
		t.Fatalf("Serve 定义数 = %d", len(defs))
	}
	d := defs[0]
	if d.Kind != "func" || !strings.Contains(d.Recv, "Handler") ||
		d.File != "api.go" || d.Line != 8 {
		t.Fatalf("Serve 定义错误: %+v", d)
	}
	if !strings.Contains(d.Signature, "Serve(w Writer) error") {
		t.Fatalf("签名错误: %q", d.Signature)
	}
	// 类型与变量
	if len(idx.Definitions("Handler")) != 1 || idx.Definitions("Handler")[0].Kind != "type" {
		t.Fatalf("Handler 定义错误: %+v", idx.Definitions("Handler"))
	}
	if len(idx.Definitions("Writer")) != 1 {
		t.Fatalf("跨文件类型定义丢失: %+v", idx.Definitions("Writer"))
	}
	// 大小写不敏感查找；同名不同 Kind（type errNil / var ErrNil）小写同键共存
	if len(idx.Definitions("errnil")) < 1 {
		t.Fatal("应按小写名索引")
	}
	if len(idx.Definitions("NoSuchSym")) != 0 {
		t.Fatal("未知符号应为空")
	}
}

func TestNewIndexReferences(t *testing.T) {
	idx, err := NewIndex(fixture(t), 100)
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.References("Handler", 50)
	found := false
	for _, r := range refs {
		if r.File == "main.go" && r.Line == 6 {
			found = true
		}
		// 定义行自身不算引用
		if r.File == "api.go" && r.Line == 6 {
			t.Fatalf("定义行不应计为引用: %+v", r)
		}
	}
	if !found {
		t.Fatalf("应找到 main.go:6 的跨文件引用: %+v", refs)
	}
	if len(idx.References("Handler", 1)) != 1 {
		t.Fatal("limit 未生效")
	}
}

func TestTouchedByLines(t *testing.T) {
	idx, err := NewIndex(fixture(t), 100)
	if err != nil {
		t.Fatal(err)
	}
	touched := idx.TouchedByLines("api.go", 11, 15)
	found := false
	for _, s := range touched {
		if s.Name == "Serve" {
			found = true
		}
	}
	if !found {
		t.Fatalf("11-15 行变更应触达 Serve（8-14 行,按区间重叠）: %+v", touched)
	}
	if len(idx.TouchedByLines("api.go", 100, 200)) != 0 {
		t.Fatal("未触达任何符号应返回空")
	}
}

// 坏语法与 vendor 文件跳过；解析失败的文件不致命。
func TestNewIndexSkips(t *testing.T) {
	idx, err := NewIndex(fixture(t), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Definitions("Helper")) != 0 {
		t.Fatal("vendor 下不应索引")
	}
	if idx.SkippedFiles() == 0 {
		t.Fatal("坏语法文件应计入 skipped")
	}
}

// maxFiles 封顶防超大仓炸内存。
func TestNewIndexMaxFiles(t *testing.T) {
	idx, err := NewIndex(fixture(t), 2)
	if err != nil {
		t.Fatal(err)
	}
	if idx.IndexedFiles() > 2 {
		t.Fatalf("indexed = %d, want <= 2", idx.IndexedFiles())
	}
}

// TestDefLineKeyNoCollision 回归（#63）：定义行去重键在两个溢出边界上不得
// 碰撞——旧 int32 (fid<<20)|line 下 fid=4096 键恒等 fid=0（fid 位溢出）、
// line=2^20+4 环绕进 fid 位与相邻文件 line=4 同键。
func TestDefLineKeyNoCollision(t *testing.T) {
	if defLineKey(0, 5) == defLineKey(4096, 5) {
		t.Fatal("fid=0 与 fid=4096 同行号键应不同")
	}
	if defLineKey(1, 1<<20+4) == defLineKey(2, 4) {
		t.Fatal("line=2^20+4 与相邻文件 line=4 键应不同")
	}
	if defLineKey(7, 9) != defLineKey(7, 9) {
		t.Fatal("同 (fid,line) 应同键")
	}
	if defLineKey(0, 1<<20+4) == defLineKey(0, 4) {
		t.Fatal("同文件 line=2^20+4 与 line=4 键应不同")
	}
}

// names4097 生成 4097 个有序文件名（fid 0..4096，覆盖 int32 键的 12 位边界）。
func names4097() []string {
	names := make([]string, 4097)
	for i := range names {
		names[i] = fmt.Sprintf("f%04d.go", i)
	}
	return names
}

// TestReferencesFileIDBeyond4096 回归（#63）：fid=4096 的定义行键在旧 int32
// 键下恒等 fid=0——文件 4096 与文件 0 定义同名符号时，文件 0 里的真引用被
// 误判为定义行剔除，References 静默漏报。
func TestReferencesFileIDBeyond4096(t *testing.T) {
	names := names4097()
	x := &Index{
		defs:      map[string][]Symbol{"baz": {{Name: "Baz", Kind: "var", File: names[4096], Line: 5}}},
		refs:      map[string][]refEnt{"Baz": {{file: 0, line: 5}}},
		fileNames: names,
	}
	got := x.References("Baz", 0)
	if len(got) != 1 || got[0].File != names[0] || got[0].Line != 5 {
		t.Fatalf("fid=4096 的定义不应误剔 fid=0 的引用: %+v", got)
	}
}

// TestGoCallersFileIDBeyond4096 回归（#63）：GoCallers 与 References 同用
// defLineKey 剔定义行——fid 边界上调用位不得被误剔。
func TestGoCallersFileIDBeyond4096(t *testing.T) {
	names := names4097()
	x := &Index{
		defs:      map[string][]Symbol{"baz": {{Name: "Baz", Kind: "func", File: names[4096], Line: 3}}},
		calls:     map[string][]refEnt{"Baz": {{file: 0, line: 3}}},
		fileNames: names,
	}
	got := x.GoCallers("Baz", 0)
	if len(got) != 1 || got[0].File != names[0] || got[0].Line != 3 {
		t.Fatalf("fid=4096 的定义不应误剔 fid=0 的调用位: %+v", got)
	}
}

// TestPreciseCallersFileIDBeyond4096 回归（#63）：PreciseCallers 同用
// defLineKey 剔定义行——fid 边界上调用位不得被误剔。
func TestPreciseCallersFileIDBeyond4096(t *testing.T) {
	names := names4097()
	x := &Index{
		defs:      map[string][]Symbol{"baz": {{Name: "Baz", Kind: "func", File: names[4096], Line: 4}}},
		calls:     map[string][]refEnt{"Baz": {{file: 0, line: 4}}},
		fileNames: names,
		files:     []fileEntry{{rel: names[0], src: []byte("package p\n\nfunc cal() {\n\tBaz()\n}\n")}},
	}
	got := x.PreciseCallers("Baz", 0)
	if len(got) != 1 || got[0].File != names[0] || got[0].Line != 4 || got[0].Caller != "cal" || got[0].Callee != "Baz" {
		t.Fatalf("fid=4096 的定义不应误剔 fid=0 的调用位: %+v", got)
	}
}

// TestReferencesLineBeyond2Pow20 回归（#63）端到端：单文件超 2^20 行时旧
// int32 键的行号位环绕进 fid 位——a.go 定义行（2^20+4）的键恰等 b.go 第 4 行
// 引用的键，真引用被误判为定义行剔除。
func TestReferencesLineBeyond2Pow20(t *testing.T) {
	dir := t.TempDir()
	// a.go：行 1 package + 2^20+2 空行 + 行 1048580（=2^20+4）定义，约 1MB < 8MB 上限
	var a strings.Builder
	a.WriteString("package p\n")
	for i := 0; i < 1<<20+2; i++ {
		a.WriteByte('\n')
	}
	a.WriteString("var Foo = 1\n")
	mustWrite(t, filepath.Join(dir, "a.go"), a.String())
	// b.go：第 4 行引用 Foo（旧键 1<<20|4 == 0<<20|(2^20+4) 碰撞）
	mustWrite(t, filepath.Join(dir, "b.go"), "package p\n\n\nfunc bar() { _ = Foo }\n")
	idx, err := NewIndex(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	defs := idx.Definitions("Foo")
	if len(defs) != 1 || defs[0].File != "a.go" || defs[0].Line != 1<<20+4 {
		t.Fatalf("定义应落在 2^20+4 行: %+v", defs)
	}
	refs := idx.References("Foo", 50)
	if len(refs) != 1 || refs[0].File != "b.go" || refs[0].Line != 4 {
		t.Fatalf("行号位环绕不应误剔 b.go:4 的引用: %+v", refs)
	}
}

// TestReferencesFileIDBeyond4096E2E 回归（#63）端到端：NewIndex(dir, 5000)——
// issue 场景原形（调用方传 maxFiles>4096），fid=4096 文件的定义不得误剔
// fid=0 文件同行的引用。
func TestReferencesFileIDBeyond4096E2E(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i <= 4096; i++ {
		switch i {
		case 0:
			mustWrite(t, filepath.Join(dir, fmt.Sprintf("f%04d.go", i)), "package p\n\nvar _ = Baz\n")
		case 4096:
			mustWrite(t, filepath.Join(dir, fmt.Sprintf("f%04d.go", i)), "package p\n\nvar Baz = 1\n")
		default:
			mustWrite(t, filepath.Join(dir, fmt.Sprintf("f%04d.go", i)), "package p\n")
		}
	}
	idx, err := NewIndex(dir, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if idx.IndexedFiles() != 4097 {
		t.Fatalf("应索引全部 4097 文件: %d", idx.IndexedFiles())
	}
	if len(idx.Definitions("Baz")) != 1 {
		t.Fatalf("Baz 定义应唯一: %+v", idx.Definitions("Baz"))
	}
	refs := idx.References("Baz", 50)
	if len(refs) != 1 || refs[0].File != "f0000.go" || refs[0].Line != 3 {
		t.Fatalf("fid=4096 的定义不应误剔 f0000.go:3 的引用: %+v", refs)
	}
}
