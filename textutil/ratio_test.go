package textutil

import "testing"

// 折算系数可配：固定测试向量（纯 ASCII / 纯 CJK / 中英混合三组）。
//
// 口径：默认系数 CJK 2 字/token、ASCII 4 字符/token，逐段向上取整。
func TestEstTokensWithRatio(t *testing.T) {
	r := DefaultCJKRatio()
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"空串", "", 0},
		// 11 个非 CJK 字符（字母+空格）→ 一段 → ceil(11/4)=3
		{"纯 ASCII", "hello world", 3},
		// 8 个汉字 → 一段 → ceil(8/2)=4
		{"纯 CJK", "诊断结论完整覆盖", 4},
		// 三段：2 CJK / 6 非 CJK（含两个空格）/ 2 CJK → 1 + 2 + 1 = 4
		{"中英混合", "检测 disk 占用", 4},
		// 单个 ASCII 字符也至少 1 token（非空下限 1）
		{"单字符", "a", 1},
		// 单个汉字：ceil(1/2)=1
		{"单汉字", "磁", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EstTokensWithRatio(c.in, r); got != c.want {
				t.Fatalf("EstTokensWithRatio(%q) = %d，期望 %d", c.in, got, c.want)
			}
		})
	}
}

// 取整方向是系统性的差，不是随机误差：合并后取整会吞掉每段的零头。
func TestEstTokensWithRounding(t *testing.T) {
	r := DefaultCJKRatio()
	// 8 段（每段 1 字符）：逐段 = 8，合并 = ceil(4/2)+ceil(4/4) = 2+1 = 3
	in := "中a中a中a中a"
	if got := EstTokensWithRounding(in, r, RoundingSegment); got != 8 {
		t.Fatalf("逐段向上取整 = %d，期望 8", got)
	}
	if got := EstTokensWithRounding(in, r, RoundingTotal); got != 3 {
		t.Fatalf("总量向上取整 = %d，期望 3", got)
	}
	// 默认入口走的就是逐段
	if EstTokensWithRatio(in, r) != EstTokensWithRounding(in, r, RoundingSegment) {
		t.Fatal("EstTokensWithRatio 必须等价于逐段向上取整")
	}
}

// 系数可配：换系数就该换结果，且非法系数不 panic（回退默认）。
func TestEstTokensRatioConfigurable(t *testing.T) {
	in := "检测 disk 占用"
	// 每字符 1 token → 10 个字符 10 token
	if got := EstTokensWithRatio(in, CJKRatio{CJKCharsPerToken: 1, ASCIICharsPerToken: 1}); got != 10 {
		t.Fatalf("1:1 系数 = %d，期望 10", got)
	}
	// CJK 1 字/token、ASCII 2 字符/token
	got := EstTokensWithRatio(in, CJKRatio{CJKCharsPerToken: 1, ASCIICharsPerToken: 2})
	// 段：2 CJK → 2；6 非 CJK → 3；2 CJK → 2 ⇒ 7
	if got != 7 {
		t.Fatalf("自定义系数 = %d，期望 7", got)
	}
	// 非法系数（<=0）回退默认，不 panic
	if EstTokensWithRatio(in, CJKRatio{}) != EstTokensWithRatio(in, DefaultCJKRatio()) {
		t.Fatal("非法系数必须回退到默认系数")
	}
}

// 已知 rune 数的折算：免字符串物化。
func TestEstTokensCJKRunesWithRatio(t *testing.T) {
	r := DefaultCJKRatio()
	if got := EstTokensCJKRunesWithRatio(10, r); got != 5 { // ceil(10/2)
		t.Fatalf("rune 折算 = %d，期望 5", got)
	}
	if got := EstTokensCJKRunesWithRatio(0, r); got != 0 {
		t.Fatalf("0 rune = %d，期望 0", got)
	}
	if got := EstTokensCJKRunesWithRatio(-3, r); got != 0 {
		t.Fatalf("负 rune = %d，期望 0", got)
	}
}

// 老接口不许被悄悄改掉：EstTokensCJK 的签名与行为保持原样。
func TestEstTokensCJKUnchanged(t *testing.T) {
	if got := EstTokensCJK("诊断结论完整覆盖"); got != 6 {
		t.Fatalf("EstTokensCJK 行为漂移，得到 %d（期望 6）", got)
	}
	if got := EstTokensCJK("hello world!!"); got != 5 {
		t.Fatalf("EstTokensCJK 行为漂移，得到 %d（期望 5）", got)
	}
	if got := EstTokensCJK(""); got != 0 {
		t.Fatalf("EstTokensCJK(\"\") = %d，期望 0", got)
	}
}

// 两套口径**不等价**，且默认系数与 EstTokensCJK 差约 1.7 倍——
// 调用方必须显式选口径，所以这个差必须被钉住（预算类场景选错会静默挤掉内容）。
func TestEstTokensTwoRatiosDiffer(t *testing.T) {
	s := "诊断结论完整覆盖"
	old := EstTokensCJK(s)
	new := EstTokensWithRatio(s, DefaultCJKRatio())
	if old == new {
		t.Fatalf("两套口径不应等价（旧=%d 新=%d）", old, new)
	}
	// 旧口径 6、新口径 4，比值 ≈ 1.5（量级对，精确比值不作门禁）
	if old < new || old > new*2 {
		t.Fatalf("两套口径的差应在 1~2 倍之间（旧=%d 新=%d）", old, new)
	}
}
