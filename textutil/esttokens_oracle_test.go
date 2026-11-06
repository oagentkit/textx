package textutil

// 两套 token 估算口径的**差异倍率**必须是可判定的量（textx#30）。
//
// 文档里写「约 2 倍」是散文，散文会漂：改了其中一个函数的除数或 +1，注释照旧
// 写着 2 倍，而实际比值已经变了。既然跨模块预算记账靠的就是这个倍率关系
// （llm 用 EstTokensCJK、promptx 曾用 EstTokens，同一条链上两种口径混用必然
// 记错账），就把它钉成断言。

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestEstTokensOracleRatioOnPureCJK 纯中文文本上 CJK 口径约为纯 rune 口径的
// 2 倍——这是文档声明的量纲差，不是精度差。
//
// 判别式取宽松区间而非等号：两个函数各有自己的取整与 +1，小样本上比值有抖动；
// 但**「接近 2」与「接近 1」是两种口径的分水岭**，区间取得足够窄就能判别
// 「有人把 CJK 口径改回纯 rune」这一类回归。
func TestEstTokensOracleRatioOnPureCJK(t *testing.T) {
	for _, n := range []int{300, 600, 1500} {
		s := strings.Repeat("汉", n)
		if got := utf8.RuneCountInString(s); got != n {
			t.Fatalf("前置：应构造 %d 个汉字，实得 %d", n, got)
		}
		plain := EstTokensOf(s)
		cjk := EstTokensCJK(s)
		ratio := float64(cjk) / float64(plain)
		if ratio < 1.8 || ratio > 2.4 {
			t.Fatalf("%d 个汉字：纯 rune 口径 %d、CJK 口径 %d，比值 %.2f ——"+
				"文档声明约 2 倍；比值落到 1 附近说明 CJK 口径被改回了纯 rune",
				n, plain, cjk, ratio)
		}
	}
}

// TestEstTokensOracleAgreeOnASCII 成对对照：纯 ASCII 上与 rune 计数正相关，
// 差异只剩「非空 +1」的量级——两条口径在英文场景几乎重合，所以中文场景的
// 2 倍差**不能**被「反正估不准」糊过去。
func TestEstTokensOracleAgreeOnASCII(t *testing.T) {
	for _, n := range []int{300, 600, 1500} {
		s := strings.Repeat("a", n)
		plain := EstTokensOf(s)
		cjk := EstTokensCJK(s)
		if diff := cjk - plain; diff < -1 || diff > 2 {
			t.Fatalf("%d 个 ASCII：纯 rune %d、CJK %d，差 %d ——"+
				"纯 ASCII 上两口径应几乎重合（仅取整与 +1 的差）", n, plain, cjk, diff)
		}
	}
}

// TestEstTokensCJKRunesMatchesCJK CJK 的「已知 rune 数」变体必须与字符版同口径
// （它存在的理由就是免字符串物化，不是另一套算法）。
func TestEstTokensCJKRunesMatchesCJK(t *testing.T) {
	for _, n := range []int{1, 7, 100, 1500} {
		s := strings.Repeat("汉", n)
		if got, want := EstTokensCJKRunes(n), EstTokensCJK(s); got != want {
			t.Fatalf("%d 个汉字：EstTokensCJKRunes = %d, EstTokensCJK = %d ——"+
				"两处口径分叉时没人答得上来「为什么这里 2 那里 3」", n, got, want)
		}
	}
}
