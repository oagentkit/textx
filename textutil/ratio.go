package textutil

// token 折算系数（可配）——与 EstTokensCJK 的关系见下。
//
// ⚠️ 两套口径并存，**不要混用**：
//
//	EstTokensCJK        固定口径（CJK 2 单位、其余 1 单位，除数写死 3），
//	                    等价于「CJK ≈ 1.5 字/token、ASCII ≈ 3 字符/token」。
//	                    **签名与行为保持不变**，老调用点不受影响。
//	EstTokensWithRatio  可配口径（本文件），默认系数 CJK 2 字/token、ASCII 4 字符/token。
//
// 为什么默认值与 EstTokensCJK 差约 1.7 倍：EstTokensCJK 是「单位数 / 3 + 1」的
// 经验折算，除数写死在函数里、不可配；而可配口径取的是 octop-memory 那一档
// （CJK 每 2 字符 1 token、ASCII 每 4 字符 1 token）。**预算类场景必须显式选口径**——
// 估错了不崩，但会静默地挤掉别的内容（同一段 1500 token 的预算，
// 换口径后能塞进的内容能差近四成）。
//
// 与真实分词器的偏差量级：默认系数下约 ±10~20%（经验值，不保证）。
// 精确计量始终归 provider usage 回传，本文件只做**事前估算**。

// CJKRatio token 折算系数：多少字符折算成 1 个 token。
//
// 非法值（<=0）在计算时回退到 DefaultCJKRatio 对应项——除零是硬错误，
// 而「系数配错」是可预期的配置问题，不该让调用方拿 panic 兜。
type CJKRatio struct {
	// CJKCharsPerToken CJK 字符数 / token（如 2 = CJK 每 2 字符 1 token）。
	CJKCharsPerToken int
	// ASCIICharsPerToken 其余字符数 / token（如 4 = ASCII 每 4 字符 1 token）。
	ASCIICharsPerToken int
}

// DefaultCJKRatio 默认折算系数：CJK 每 2 字符 1 token、ASCII 每 4 字符 1 token。
func DefaultCJKRatio() CJKRatio {
	return CJKRatio{CJKCharsPerToken: 2, ASCIICharsPerToken: 4}
}

// normalized 补齐非法系数（<=0 → 默认值对应项）。
func (r CJKRatio) normalized() CJKRatio {
	d := DefaultCJKRatio()
	if r.CJKCharsPerToken <= 0 {
		r.CJKCharsPerToken = d.CJKCharsPerToken
	}
	if r.ASCIICharsPerToken <= 0 {
		r.ASCIICharsPerToken = d.ASCIICharsPerToken
	}
	return r
}

// TokenRounding 折算取整方向。
//
// 两种取整在**混合文本**上差系统性：把同类字符合并后再取整，会把每段的零头
// 一起吞掉——多段混合文本累加起来是成规模的低估，不是随机误差。
type TokenRounding int

const (
	// RoundingSegment 逐段向上取整（默认）：按**连续同类字符游程**切段，
	// 每段各自向上取整后求和。
	//
	// 「中a中a中a中a」在逐段口径下是 8 段（每段 1 字符）→ 8 token；
	// 合并口径下只有 3 token。差的正是那 5 个「半截 token」。
	RoundingSegment TokenRounding = iota
	// RoundingTotal 总量向上取整：同类字符合计后各向上取整一次再求和。
	RoundingTotal
)

// EstTokensWithRatio 用指定折算系数估算 token（逐段向上取整）。
//
// 折算口径见 CJKRatio 的说明；默认系数用 DefaultCJKRatio()。
// 空串返回 0；非空串至少 1。
func EstTokensWithRatio(s string, r CJKRatio) int {
	return EstTokensWithRounding(s, r, RoundingSegment)
}

// EstTokensWithRounding 用指定折算系数与取整方向估算 token。
//
// 取整方向是**参数**而不是另开一个函数：两种口径下「哪个数才是这个文本的
// token 数」是同一个问题的两个答案，分成两个函数只会逼调用方在两套命名之间猜。
//
// 空串返回 0；非空串至少 1。
func EstTokensWithRounding(s string, r CJKRatio, mode TokenRounding) int {
	if s == "" {
		return 0
	}
	r = r.normalized()

	if mode == RoundingTotal {
		cjk, other := 0, 0
		for _, ru := range s {
			if isCJKUnit(ru) {
				cjk++
			} else {
				other++
			}
		}
		return ceilDiv(cjk, r.CJKCharsPerToken) + ceilDiv(other, r.ASCIICharsPerToken)
	}

	// 逐段：连续同类字符算一段，每段各自向上取整。
	total := 0
	runLen, runCJK := 0, false
	for _, ru := range s {
		cjk := isCJKUnit(ru)
		if runLen == 0 {
			runCJK = cjk
			runLen = 1
			continue
		}
		if cjk != runCJK {
			total += segmentTokens(runLen, runCJK, r)
			runCJK, runLen = cjk, 1
			continue
		}
		runLen++
	}
	if runLen > 0 {
		total += segmentTokens(runLen, runCJK, r)
	}
	return total
}

// EstTokensCJKRunesWithRatio 已知 rune 数且内容以 CJK 为主时的折算估算
// （免字符串物化——compact 等调用方手里只有 rune 计数中间量）。
//
// 与 EstTokensCJKRunes 同口径：**按「全部字符都是 CJK」估算**。
// 文本实际以 ASCII 为主时这个值会偏大，调用方需自行判断是否适用。
//
// 非法系数回退到默认系数；n<=0 返回 0。
func EstTokensCJKRunesWithRatio(n int, r CJKRatio) int {
	if n <= 0 {
		return 0
	}
	r = r.normalized()
	return ceilDiv(n, r.CJKCharsPerToken)
}

// segmentTokens 一段同类字符的 token 数（向上取整）。
func segmentTokens(n int, cjk bool, r CJKRatio) int {
	if cjk {
		return ceilDiv(n, r.CJKCharsPerToken)
	}
	return ceilDiv(n, r.ASCIICharsPerToken)
}

// isCJKUnit 该 rune 是否按 CJK（宽字符）计量。
//
// 判据与 EstTokensCJK 同源（runeUnitsCJK == 2），**故意不另立一套**：
// 两处口径分叉时，「为什么这里算 1 个 token 那里算 2 个」没人答得上来。
func isCJKUnit(r rune) bool {
	return runeUnitsCJK(r) == 2
}

// ceilDiv 向上取整除法（分母 > 0 由调用方保证）。
func ceilDiv(a, b int) int {
	if a <= 0 {
		return 0
	}
	return (a + b - 1) / b
}
