package reportutil

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		input string
		want  string
		ok    bool
	}{
		{"critical", Critical, true},
		{"CRITICAL", Critical, true},
		{"high", High, true},
		{"HIGH", High, true},
		{"严重", High, true},
		{"medium", Medium, true},
		{"MEDIUM", Medium, true},
		{"warning", Medium, true},
		{"low", Low, true},
		{"info", Low, true},
		{"unknown", Low, false},
		{"", Low, false},
	}
	for _, c := range cases {
		got, ok := Normalize(c.input)
		if got != c.want || ok != c.ok {
			t.Errorf("Normalize(%q) = (%q, %v), want (%q, %v)", c.input, got, ok, c.want, c.ok)
		}
	}
}

func TestRank(t *testing.T) {
	// 四档刻度 4/3/2/1（oagentkit/textx#20）：顶格档独立于 high。
	if Rank(Critical) != 4 || Rank(High) != 3 || Rank(Medium) != 2 || Rank(Low) != 1 {
		t.Errorf("刻度应为 4/3/2/1: %d/%d/%d/%d",
			Rank(Critical), Rank(High), Rank(Medium), Rank(Low))
	}
	if Rank(High) <= Rank(Medium) {
		t.Error("high 应比 medium 严重")
	}
	if Rank(Medium) <= Rank(Low) {
		t.Error("medium 应比 low 严重")
	}
}

func TestHigher(t *testing.T) {
	if Higher(Critical, High) != Critical {
		t.Error("Higher(critical, high) 应为 critical")
	}
	if Higher(High, Medium) != High {
		t.Error("Higher(high, medium) 应为 high")
	}
	if Higher(Low, High) != High {
		t.Error("Higher(low, high) 应为 high")
	}
	if Higher(Medium, Medium) != Medium {
		t.Error("Higher(medium, medium) 应为 medium")
	}
}

func TestValid(t *testing.T) {
	if !Valid(Critical) || !Valid(High) || !Valid(Medium) || !Valid(Low) {
		t.Error("critical/high/medium/low 应有效")
	}
	if Valid("p0") || Valid("") {
		t.Error("别名原词/空 应无效（Valid 只认钉死词表，别名走 Normalize）")
	}
}

// 别名词表：外部审查专家常用 P0/nit 等通用研发词表。四档化（v0.8.0）后
// P0-P3 按 4 档刻度取自然映射，fatal/blocker 等顶格词归 critical。
func TestNormalizeAliases(t *testing.T) {
	critical := map[string]string{"P0": Critical, "fatal": Critical, "BLOCKER": Critical, "致命": Critical}
	for in, want := range critical {
		if got, ok := Normalize(in); got != want || !ok {
			t.Fatalf("Normalize(%q)=%q,%v，want %q,true", in, got, ok, want)
		}
	}
	high := map[string]string{"P1": High, "URGENT": High, "major": High}
	for in, want := range high {
		if got, ok := Normalize(in); got != want || !ok {
			t.Fatalf("Normalize(%q)=%q,%v，want %q,true", in, got, ok, want)
		}
	}
	if got, ok := Normalize("p2"); got != Medium || !ok {
		t.Fatalf("p2 应 medium: %q %v", got, ok)
	}
	for _, in := range []string{"p3", "nit", "nits", "trivial"} {
		if got, ok := Normalize(in); got != Low || !ok {
			t.Fatalf("%q 应 low（合法词）: %q %v", in, got, ok)
		}
	}
	// 原词表行为：critical 不再折进 high，顶格档独立
	if got, ok := Normalize("critical"); got != Critical || !ok {
		t.Fatalf("critical 应 critical: %q %v", got, ok)
	}
	if _, ok := Normalize("gibberish"); ok {
		t.Fatal("越界词仍应 false")
	}
}

// Normalize 输出恒属钉死词表——下游可安全 switch/查表，不需要第四分支特判。
func TestNormalizeOutputAlwaysInVocab(t *testing.T) {
	for _, in := range []string{"critical", "fatal", "p0", "blocker", "high", "p1",
		"medium", "p2", "low", "nit", "error", "严重", "", "gibberish", "高危"} {
		got, _ := Normalize(in)
		if !Valid(got) {
			t.Errorf("Normalize(%q) = %q 不在词表内", in, got)
		}
	}
}

// Normalize3 三档兼容钩子：critical 折进 high，其余档位与 Normalize 一致；
// 越界词的 bool 语义不变。
func TestNormalize3FoldsCritical(t *testing.T) {
	if got, ok := Normalize3("critical"); got != High || !ok {
		t.Fatalf("Normalize3(critical)=%q,%v，want %q,true", got, ok, High)
	}
	for in, want := range map[string]string{"P0": High, "fatal": High, "阻断": High,
		"high": High, "medium": Medium, "p2": Medium, "low": Low, "nit": Low} {
		if got, ok := Normalize3(in); got != want || !ok {
			t.Fatalf("Normalize3(%q)=%q,%v，want %q,true", in, got, ok, want)
		}
	}
	if _, ok := Normalize3("gibberish"); ok {
		t.Fatal("越界词仍应 false")
	}
	// 输出恒属词表（三档消费点可安全 switch）
	for _, in := range []string{"critical", "p0", "high", "medium", "low", "", "gibberish"} {
		if got, _ := Normalize3(in); got != High && got != Medium && got != Low {
			t.Errorf("Normalize3(%q) = %q 越出三档", in, got)
		}
	}
}
