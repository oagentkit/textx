package gotyped

// extractor_test.go —— 精确调用边全拓扑断言（#106）：同名跨包函数区分、
// 方法端点 Recv、包级函数、根包空 Dir、失败语义（无 go.mod/超上限）。

import (
	"context"
	"path/filepath"
	"testing"
)

func edgeKey(e Edge) [2]string { return [2]string{e.From.String(), e.To.String()} }

// TestEdgesProbeTopology testdata/probe 已知拓扑全量断言：词法面会混淆的
// 同名跨包函数（lib.Helper vs util.Helper）在 typed 面必须精确分边。
func TestEdgesProbeTopology(t *testing.T) {
	dir := filepath.Join("testdata", "probe")
	edges, err := New().Edges(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[[2]string]bool{}
	for _, e := range edges {
		got[edgeKey(e)] = true
	}
	want := [][2]string{
		{"main", "lib.Greet"},
		{"main", "lib.Helper"},
		{"main", "lib.Widget.Run"},
		{"main", "util.Helper"},
		{"lib.Greet", "util.Helper"},
		{"lib.Widget.Run", "util.Helper"},
	}
	for _, w := range want {
		if !got[w] {
			t.Fatalf("缺边 %v → %v（全量 %v）", w[0], w[1], edges)
		}
		delete(got, w)
	}
	if len(got) != 0 {
		t.Fatalf("多余边: %v", got)
	}
}

// TestEdgesNoModule 无 go.mod 目录必须报错（调用方据此降级词法面）。
func TestEdgesNoModule(t *testing.T) {
	if _, err := New().Edges(context.Background(), "testdata"); err == nil {
		t.Fatal("无 go.mod 应返回 error")
	}
}

// TestEdgesFileLimit 文件上限：小 MaxFiles 对 probe（3 文件）应触发超限失败。
func TestEdgesFileLimit(t *testing.T) {
	x := &Extractor{MaxFiles: 1}
	if _, err := x.Edges(context.Background(), filepath.Join("testdata", "probe")); err == nil {
		t.Fatal("超上限应返回 error")
	}
}

// TestSymbolString 端点键文本形态。
func TestSymbolString(t *testing.T) {
	cases := []struct {
		s    Symbol
		want string
	}{
		{Symbol{Dir: "", Recv: "", Name: "main"}, "main"},
		{Symbol{Dir: "lib", Recv: "Widget", Name: "Run"}, "lib.Widget.Run"},
		{Symbol{Dir: "a/b", Recv: "", Name: "F"}, "a/b.F"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Fatalf("Symbol%v.String()=%q, want %q", c.s, got, c.want)
		}
	}
}
