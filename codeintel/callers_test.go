package codeintel

import (
	"path/filepath"
	"testing"
)

// callerFixture 精化调用位 fixture：同包调用/跨包调用/方法调用/递归/
// 字符串与注释假命中源/包级 var 初始化器调用。
func callerFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/cx\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, "core.go"), `package cx

// Foo 被测函数（导出，递归形态）。
func Foo(n int) int {
	if n <= 1 {
		return 1
	}
	return n * Foo(n-1)
}

// Item 被测类型。
type Item struct{ N int }

// Grow 被测方法。
func (i Item) Grow() Item { i.N++; return i }

// shrink 未导出方法。
func (i Item) shrink() Item { return i }
`)
	mustWrite(t, filepath.Join(dir, "user.go"), `package cx

// UseFoo 同包调用方。
func UseFoo() int {
	// Foo( 在注释里——token 候选层就该排除，AST 层复核兜底
	s := "Fake( 假命中"
	_ = s
	return Foo(5)
}

var table = map[string]int{"a": Foo(1)}
`)
	mustWrite(t, filepath.Join(dir, "ext", "ext.go"), `package ext

import "example.com/cx"

// Wrap 跨包调用方（限定名 + 方法调用）。
func Wrap() int {
	i := cx.Item{N: 1}
	_ = i.Grow()
	_ = i.shrink()
	return cx.Foo(3)
}
`)
	return dir
}

// TestPreciseCallersBasics 基础精化：命中行、Caller/Callee、定义行剔除。
func TestPreciseCallersBasics(t *testing.T) {
	x, err := NewIndex(callerFixture(t), 100)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	refs := x.PreciseCallers("Foo", 0)
	if len(refs) == 0 {
		t.Fatal("无命中")
	}
	want := map[string]bool{
		"user.go:8":     false, // UseFoo 内调用
		"user.go:11":    false, // var 初始化器
		"core.go:8":     false, // 递归（定义行之外）
		"ext/ext.go:10": false, // cx.Foo 限定调用
	}
	for _, r := range refs {
		key := r.File + ":" + itoaCaller(r.Line)
		if _, ok := want[key]; !ok {
			t.Fatalf("意外命中 %s（%+v）", key, r)
		}
		want[key] = true
		if r.Callee == "" {
			t.Fatalf("%s 缺 Callee", key)
		}
	}
	for k, hit := range want {
		if !hit {
			t.Fatalf("缺命中 %s（got %+v）", k, refs)
		}
	}
	// 封闭函数与限定名（File:Line 键——跨文件同行号会撞）
	byKey := map[string]CallerRef{}
	for _, r := range refs {
		byKey[r.File+":"+itoaCaller(r.Line)] = r
	}
	if got := byKey["user.go:8"]; got.Caller != "UseFoo" || got.Callee != "Foo" {
		t.Fatalf("user.go:8 = %+v", got)
	}
	if got := byKey["user.go:11"]; got.Caller != CallerInit {
		t.Fatalf("var 初始化器 Caller = %+v", got)
	}
	if got := byKey["core.go:8"]; got.Caller != "Foo" {
		t.Fatalf("递归调用 Caller = %+v", got)
	}
	if got := byKey["ext/ext.go:10"]; got.File != "ext/ext.go" || got.Caller != "Wrap" || got.Callee != "cx.Foo" {
		t.Fatalf("ext 调用 = %+v", got)
	}
}

// TestPreciseCallersMethod 方法调用：裸名命中，Caller 给到接收者限定形态；
// 同名异接收者如实全命中（边界声明，不做类型归属）。
func TestPreciseCallersMethod(t *testing.T) {
	x, err := NewIndex(callerFixture(t), 100)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	refs := x.PreciseCallers("Grow", 0)
	if len(refs) != 1 || refs[0].File != "ext/ext.go" || refs[0].Caller != "Wrap" || refs[0].Callee != "i.Grow" {
		t.Fatalf("Grow 命中 = %+v", refs)
	}
	// 定义行不计：core.go 上的 Grow 声明不出现在结果里
	for _, r := range refs {
		if r.File == "core.go" {
			t.Fatalf("定义行泄漏: %+v", refs)
		}
	}
}

// TestPreciseCallersNoFalsePositive 字符串/注释形态不产命中（token 候选层排除，
// AST 层复核双保险）。
func TestPreciseCallersNoFalsePositive(t *testing.T) {
	x, err := NewIndex(callerFixture(t), 100)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	for _, q := range []string{"Fake", "UseFoo2", "不存在的名字"} {
		if got := x.PreciseCallers(q, 0); len(got) != 0 {
			t.Fatalf("%s 应零命中, got %+v", q, got)
		}
	}
}

// TestPreciseCallersLimit 封顶与 nil 安全。
func TestPreciseCallersLimit(t *testing.T) {
	x, err := NewIndex(callerFixture(t), 100)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	if got := x.PreciseCallers("Foo", 2); len(got) != 2 {
		t.Fatalf("limit=2 命中 %d", len(got))
	}
	var nilIdx *Index
	if got := nilIdx.PreciseCallers("Foo", 0); got != nil {
		t.Fatalf("nil 索引应返回 nil, got %+v", got)
	}
	// 有序性：(File,Line) 升序
	refs := x.PreciseCallers("Foo", 0)
	for i := 1; i < len(refs); i++ {
		if refs[i-1].File > refs[i].File ||
			(refs[i-1].File == refs[i].File && refs[i-1].Line > refs[i].Line) {
			t.Fatalf("乱序: %+v", refs)
		}
	}
}

// TestPreciseCallersVsGoCallers 与 GoCallers 同源对照：候选集一致（定义行剔除后），
// 精化只减不增——上游契约钉死，防两查询漂移。
func TestPreciseCallersVsGoCallers(t *testing.T) {
	x, err := NewIndex(callerFixture(t), 100)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	lex := x.GoCallers("Foo", 0)
	precise := x.PreciseCallers("Foo", 0)
	if len(precise) > len(lex) {
		t.Fatalf("精化多于词法候选: 精化 %d > 词法 %d", len(precise), len(lex))
	}
	seen := map[string]bool{}
	for _, r := range lex {
		seen[r.File+":"+itoaCaller(r.Line)] = true
	}
	for _, r := range precise {
		if !seen[r.File+":"+itoaCaller(r.Line)] {
			t.Fatalf("精化命中不在词法候选内: %+v", r)
		}
	}
}

func itoaCaller(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
