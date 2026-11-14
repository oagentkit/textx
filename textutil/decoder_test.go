package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// roundTrip 把 s 按任意切块喂给一个 Decoder，返回「各块输出按序拼接 + Flush」
// 与「中途每块输出是否全为合法 UTF-8」。参考实现语义的等价性判据：合法
// UTF-8 语料无论怎么切，拼回必须逐字节等于原文，且任何中间输出不含半个字符。
func roundTrip(t *testing.T, chunks [][]byte) (string, bool) {
	t.Helper()
	d := New()
	var b strings.Builder
	valid := true
	for _, c := range chunks {
		out := d.Decode(c)
		if !utf8.ValidString(out) {
			valid = false
		}
		b.WriteString(out)
	}
	b.WriteString(d.Flush())
	return b.String(), valid
}

// TestDecoderSplitEveryPosition 全切点回归：合法语料在每个字节位置切一刀，
// 两块喂入后必须逐字节还原原文（增量解码的存续语义 == 一次性解码）。
// 3 字节 CJK 被读边界拦腰切断正是 issue 点名的乱码场景。
func TestDecoderSplitEveryPosition(t *testing.T) {
	corpora := []string{
		"hello, world",
		"你好，世界",
		"aé你😀b",            // 1/2/3/4 字节字符混排
		"UTF-8 边界 你好 ok 😀", // ASCII 与多字节交错
	}
	for _, s := range corpora {
		for i := 0; i <= len(s); i++ {
			got, valid := roundTrip(t, [][]byte{[]byte(s[:i]), []byte(s[i:])})
			if got != s || !valid {
				t.Errorf("split %q at %d: got %q (valid=%v), want %q", s, i, got, valid, s)
			}
		}
	}
}

// TestDecoderByteByByte 逐字节喂入：中间输出必须恒为合法 UTF-8 且在残字符
// 未补齐前为空，最终拼接还原原文。
func TestDecoderByteByByte(t *testing.T) {
	s := "你好😀end"
	d := New()
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		out := d.Decode([]byte{s[i]})
		if !utf8.ValidString(out) {
			t.Fatalf("chunk %d: 中间输出含非法 UTF-8: %q", i, out)
		}
		b.WriteString(out)
	}
	b.WriteString(d.Flush())
	if b.String() != s {
		t.Fatalf("逐字节喂入拼回 = %q, want %q", b.String(), s)
	}
}

// TestDecoderTruncatedFlush 截断残留的 Flush 语义：残留是非法字节，与
// Decode 的非法字节策略同口径——逐字节吐 U+FFFD，不静默丢弃；二次 Flush
// 返回空。同时钉住「截断前缀 Decode 不吐字」。
func TestDecoderTruncatedFlush(t *testing.T) {
	cases := []struct {
		name  string
		chunk []byte
		want  string
	}{
		{"3字节序列残1", []byte{'\xe4'}, "\uFFFD"},
		{"3字节序列残2", []byte{'\xe4', '\xbd'}, "\uFFFD\uFFFD"},
		{"4字节序列残3", []byte{'\xf0', '\x9f', '\x98'}, "\uFFFD\uFFFD\uFFFD"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := New()
			if out := d.Decode(c.chunk); out != "" {
				t.Fatalf("截断前缀 Decode = %q, want \"\"", out)
			}
			if got := d.Flush(); got != c.want {
				t.Fatalf("Flush = %q, want %q", got, c.want)
			}
			if got := d.Flush(); got != "" {
				t.Fatalf("二次 Flush = %q, want \"\"", got)
			}
		})
	}
}

// TestDecoderCompleteFlush 完整消费后的 Flush 是空操作。
func TestDecoderCompleteFlush(t *testing.T) {
	d := New()
	if out := d.Decode([]byte("你好")); out != "你好" {
		t.Fatalf("Decode = %q", out)
	}
	if got := d.Flush(); got != "" {
		t.Fatalf("Flush = %q, want \"\"", got)
	}
}

// TestDecoderIllegalBytes 非法字节逐字节替换（消费 1 字节防死循环）：非法
// 起始字节、孤续字节、overlong、UTF-16 surrogate、>U+10FFFF 起点，各自
// 消费 1 字节吐 1 个 U+FFFD，循环必然推进。
func TestDecoderIllegalBytes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"夹在 ASCII 中", "a\xffb", "a\uFFFDb"},
		{"连续非法字节逐字节计", "\xff\xff", "\uFFFD\uFFFD"},
		{"孤续字节", "\x80abc", "\uFFFDabc"},
		{"overlong 编码", "\xc0\x80", "\uFFFD\uFFFD"},
		{"surrogate 码位", "\xed\xa0\x80", "\uFFFD\uFFFD\uFFFD"},
		{"超出码位上限", "\xf5\x80\x80\x80", "\uFFFD\uFFFD\uFFFD\uFFFD"},
		{"截断序列后跟 ASCII 不吞字", "\xe4ab", "\uFFFDab"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := New()
			if got := d.Decode([]byte(c.in)); got != c.want {
				t.Fatalf("Decode(%q) = %q, want %q", c.in, got, c.want)
			}
			if got := d.Flush(); got != "" {
				t.Fatalf("Flush = %q, want \"\"", got)
			}
		})
	}
}

// TestDecoderIllegalAcrossChunks 非法字节与截断序列跨块交错：FFFD 后的
// pending 必须继续正常解码（死循环回归——整段跳过的实现会把残字符前缀
// 一并吞掉或永久卡住）。
func TestDecoderIllegalAcrossChunks(t *testing.T) {
	d := New()
	if got := d.Decode([]byte("a\xff")); got != "a\uFFFD" {
		t.Fatalf("chunk1 = %q", got)
	}
	if got := d.Decode([]byte("\xe4\xbd\xa0")); got != "你" {
		t.Fatalf("chunk2 = %q, want 你", got)
	}
	if got := d.Flush(); got != "" {
		t.Fatalf("Flush = %q, want \"\"", got)
	}
	// 截断前缀撞上非法字节：当场按非法路径清账，不悬挂
	d2 := New()
	if got := d2.Decode([]byte("\xe4\xff")); got != "\uFFFD\uFFFD" {
		t.Fatalf("Decode(\\xe4\\xff) = %q", got)
	}
	if got := d2.Decode([]byte("\xbd\xa0")); got != "\uFFFD\uFFFD" {
		t.Fatalf("尾随孤续 = %q", got)
	}
}

// TestDecoderEmptyChunk 空 chunk 不动状态：pending 照常留存，流未结束。
func TestDecoderEmptyChunk(t *testing.T) {
	d := New()
	if got := d.Decode(nil); got != "" {
		t.Fatalf("Decode(nil) = %q", got)
	}
	if got := d.Decode([]byte("\xe4")); got != "" {
		t.Fatalf("Decode(残前缀) = %q", got)
	}
	if got := d.Decode(nil); got != "" {
		t.Fatalf("空块后 Decode(nil) = %q", got)
	}
	if got := d.Decode([]byte("\xbd\xa0")); got != "你" {
		t.Fatalf("补齐块 = %q, want 你", got)
	}
}

// TestDecoderFlushThenReuse Flush 后 Decoder 回到空状态，继续 Decode 等同新流。
func TestDecoderFlushThenReuse(t *testing.T) {
	d := New()
	_ = d.Decode([]byte("\xe4"))
	if got := d.Flush(); got != "\uFFFD" {
		t.Fatalf("Flush = %q", got)
	}
	if got := d.Decode([]byte("你好")); got != "你好" {
		t.Fatalf("Flush 后 Decode = %q", got)
	}
}

// TestDecoderStateIsolation 状态不可跨流共享：A 流的残字符前缀不得泄漏给 B 流。
func TestDecoderStateIsolation(t *testing.T) {
	a, b := New(), New()
	if got := a.Decode([]byte("\xe4\xbd")); got != "" {
		t.Fatalf("A 残前缀 Decode = %q", got)
	}
	if got := b.Decode([]byte("ok")); got != "ok" {
		t.Fatalf("B Decode = %q", got)
	}
	if got := a.Decode([]byte("\xa0")); got != "你" {
		t.Fatalf("A 补齐 = %q", got)
	}
}

// TestDecoderMatchesToValidUTF8 与 stdlib 单块语义交叉对账：整段一次性
// Decode 的输出 == strings.ToValidUTF8(s, "\\uFFFD")（孤立非法字节场景，
// 二者口径一致；连续非法字节的「run 归并」是 ToValidUTF8 特有优化，
// 本包按参考实现逐字节计，见 TestDecoderIllegalBytes）。
func TestDecoderMatchesToValidUTF8(t *testing.T) {
	for _, s := range []string{
		"plain ascii",
		"a\xffbé你😀c",
		"\xe4\xbd\xa0\xe5\xa5\xbd",
	} {
		d := New()
		got := d.Decode([]byte(s)) + d.Flush()
		if want := strings.ToValidUTF8(s, "\uFFFD"); got != want {
			t.Errorf("Decode 整段(%q) = %q, want ToValidUTF8 %q", s, got, want)
		}
	}
}
