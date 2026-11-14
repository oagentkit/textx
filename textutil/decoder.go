package textutil

import (
	"strings"
	"unicode/utf8"
)

// Decoder 增量 UTF-8 解码器：把任意切块的字节流拼装成安全文本（textx#91，
// 宿主参考实现 bianque internal/terminal/utf8stream.go）。
//
// 任何「字节流 → 文本」的消费方（PTY 输出、WebSocket 帧、串口、日志 tail）
// 都会撞同一个坑：读边界把 3 字节 CJK 字符拦腰切成两半时，对每块独立
// string(chunk) 解码即产生乱码——前一块吐出半个残字符，后一块吐出另一半。
// Decode 的对策是**不完整尾字节留存**：残字符的前缀先攒着，等下一块补齐再吐。
//
// 用法（每条流一个实例）：
//
//	d := textutil.New()
//	for chunk := range byteStream {
//	    text := d.Decode(chunk) // 随时可安全展示/落盘
//	    _ = text
//	}
//	tail := d.Flush() // 流结束：把残留吐干净
//
// # 契约（三条缺一不可，简化任何一条都会复现乱码/死循环缺陷）
//
//   - 不完整序列留存：Decode 结束时残留的 pending 恒为**至多一个**有效
//     多字节序列的前缀（FullRune=false 才留存，非法编码当场消费）；
//   - 非法字节逐字节替换：解码到非法/孤续/超长字节（含 UTF-16 surrogate
//     码位）时，每消费 1 字节吐 1 个 U+FFFD——「消费 1 字节」是防死循环的
//     关键，整段跳过会让残字符前缀永远等不到补齐；
//   - Flush 清账：把残留按非法字节同一口径逐字节吐 U+FFFD（截断的
//     序列不静默丢弃），随后 Decoder 回到空状态。
//
// 状态不可跨流共享：一个 Decoder 只服务一条字节流，且非并发安全——
// 多路流各自 New，共享会把 A 流的残字符前缀喂给 B 流。
type Decoder struct {
	pending []byte
}

// New 构造增量 UTF-8 解码器（空状态，可直接对第一条字节块调 Decode）。
func New() *Decoder { return &Decoder{} }

// Decode 喂入下一个字节块，返回当下可安全消费的文本。
//
// 块边界任意：多字节字符被切块拦腰截断时，前缀留存到 pending 等下一块补齐，
// 返回值里绝不出现半个字符；非法字节（FullRune 判定「无效但完整」的序列）
// 逐字节吐 U+FFFD 并消费 1 字节，循环必然推进、不存在卡死形态。
// 空 chunk（len=0）返回 "" 且不动 pending——零字节块不代表流结束，
// 收尾请走 Flush。
func (d *Decoder) Decode(chunk []byte) string {
	if len(chunk) == 0 {
		return ""
	}
	d.pending = append(d.pending, chunk...)
	var b strings.Builder
	for len(d.pending) > 0 {
		if !utf8.FullRune(d.pending) {
			// 尾部是不完整的多字节序列：留存等下一块
			break
		}
		r, size := utf8.DecodeRune(d.pending)
		if r == utf8.RuneError && size <= 1 {
			// 非法字节：吐替换符并消费 1 字节（防死循环）
			b.WriteRune(utf8.RuneError)
			d.pending = d.pending[1:]
			continue
		}
		b.Write(d.pending[:size])
		d.pending = d.pending[size:]
	}
	return b.String()
}

// Flush 流收尾：把残留的不完整序列逐字节替换为 U+FFFD 后清空状态，
// 返回这部分文本。解码全程无残留时返回 ""。Flush 后可继续 Decode
// （等同开启新流）。
func (d *Decoder) Flush() string {
	if len(d.pending) == 0 {
		return ""
	}
	var b strings.Builder
	for range d.pending {
		b.WriteRune(utf8.RuneError)
	}
	d.pending = d.pending[:0]
	return b.String()
}
