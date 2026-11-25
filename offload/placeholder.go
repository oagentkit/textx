// 占位文本契约：外置对象在消息历史里留下的那一小段替身文本。
//
// ## 为什么要有格式与解析两件套
//
// 占位文本是**跨轮复用的字符串**：内容寻址保证同一份内容逐轮得到逐字节相同的
// 占位文本，历史前缀因此能命中提示缓存——这正是外置收益的一半（另一半是不重发本体）。
// 一旦它可被机械解析，宿主才能在不重新计算摘要的前提下回答「这条历史上外置过什么」
// （对账、回收、审计）。故本文件与 spill 的 TruncHint/ParseHint 同纪律：格式单源，
// 改格式即改契约。
//
// ## 字段取舍
//
// 每个字段都有存在理由，缺一个就会有一类问题变成不可排查：
//
//   - kind/id：模型据此判断「这是什么东西、同一 id 是否同一份字节」；
//   - path：取回入口。**必须与真实落盘键逐字一致**，否则模型按图索骥必然落空；
//   - bytes：模型据此估算取回成本，决定要不要取；
//   - mime：决定取回后如何解释；缺失时模型只能盲解。
//
// 解析面要求各字段不含空格；path 由**写入侧 StorePath 与解析面共用的
// validRelPath** 双向净化（写入侧产出后自检、解析面收下前校验）——解析面是读入面，
// 文本可来自模型复述或被注入的外部内容，不能依赖写入侧的保证。
package offload

import (
	"strconv"
	"strings"
)

const (
	// markerPrefix 占位文本的定长前缀（解析面的定位锚）。
	markerPrefix = "[外置对象 "

	// markerSuffix 字段区结束符。
	markerSuffix = "]"

	// DefaultRetrieveHint 缺省取回指引。
	//
	// 指引不写死具体工具名：取回入口由宿主决定（本地文件、对象存储、专用工具……），
	// 内核写死某个名字只会在宿主换实现后变成误导——指向一个不存在的工具
	// 比不给指引更糟。宿主有自定义时经 Request.RetrieveHint 覆盖。
	DefaultRetrieveHint = "原文不在本条消息内；需要时按 path 取回（id 相同即字节相同）。"
)

// Ref 占位文本的可解析部分（取回指引不在契约内，由宿主自定故不解析）。
type Ref struct {
	// Kind 对象类型。
	Kind string

	// ID 摘要前缀（见 TruncatedHash）。
	ID string

	// Path 逻辑落盘键，须与实际落盘键逐字一致。
	Path string

	// Bytes 原始字节数。
	Bytes int

	// MimeType 媒体类型；空则不渲染该字段。
	MimeType string
}

// Render 生成占位文本。
//
// 缺省形态 191 字节（由 TestPlaceholderBodySmallerThanThreshold 钉住上限）——这是 MinBytes 取 4 KiB 的前提：小到占位体量的若干分之一，
// 跨过阈值后替换一定是净赚（见 offload.go 包注释「关键取舍一」）。
// 同一 (Ref, hint) 必须产出**逐字节相同**的文本，故渲染内不含时间戳、序号、
// 随机数等任何随轮变化的成分。
func Render(r Ref, hint string) string {
	h := strings.TrimSpace(hint)
	if h == "" {
		h = DefaultRetrieveHint
	}
	var b strings.Builder
	b.WriteString(markerPrefix)
	b.WriteString("kind=" + escapeFieldValue(r.Kind))
	b.WriteString(" id=" + escapeFieldValue(r.ID))
	b.WriteString(" path=" + r.Path)
	b.WriteString(" bytes=" + strconv.Itoa(r.Bytes))
	if r.MimeType != "" {
		b.WriteString(" mime=" + escapeFieldValue(r.MimeType))
	}
	b.WriteString(markerSuffix)
	b.WriteString(" ")
	b.WriteString(h)
	return b.String()
}

// Parse 解析占位文本：命中定长前缀且字段齐全才返回 true，否则返回 (零值, false)。
//
// 反向的宽松度要求与写入侧对称：宁可漏判（当作普通文本）也不误判——
// 把一段普通文本错认成外置占位会让宿主去取一个并不存在的键。
func Parse(text string) (Ref, bool) {
	i := strings.Index(text, markerPrefix)
	if i < 0 {
		return Ref{}, false
	}
	rest := text[i+len(markerPrefix):]
	j := strings.Index(rest, markerSuffix)
	if j < 0 {
		return Ref{}, false
	}
	var r Ref
	var n int
	for _, kv := range strings.Split(rest[:j], " ") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			// 字段区里的裸词是破损文本：旧实现 continue 跳过它，于是一段
			// 「kind=image large」会以 ok=true 返回 Kind=image——静默丢字段。
			// 宁可漏判（当普通文本）也不把破损文本认成合法占位。
			return Ref{}, false
		}
		switch k {
		case "kind":
			r.Kind = unescapeFieldValue(v)
		case "id":
			r.ID = unescapeFieldValue(v)
		case "path":
			r.Path = v
		case "bytes":
			b, err := strconv.Atoi(v)
			if err != nil {
				return Ref{}, false // 字节数不可解析即契约破损，不猜
			}
			r.Bytes = b
			n++
		case "mime":
			r.MimeType = unescapeFieldValue(v)
		}
	}
	if r.Kind == "" || r.ID == "" || r.Path == "" || n == 0 {
		return Ref{}, false
	}
	// 解析面是读入面：path 必须过与写入侧同一份形状判据。原样收下会让宿主按
	// Ref.Path 取回时踩到路径穿越（`..`、分隔符、空白都进来过）。
	if !validRelPath(r.Path) {
		return Ref{}, false
	}
	return r, true
}

// fieldSafe 字段值里可原样出现的字节集。空格与 '%' 不在内：空格是字段分隔符，
// '%' 是转义引导符。'=' 也不在内，免得将来解析面改按 '=' 分词时出现第二套口径。
func fieldSafe(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '.', c == '_', c == '~', c == ':', c == '/', c == '+', c == '@':
		return true
	default:
		return false
	}
}

const upperHex = "0123456789ABCDEF"

// escapeFieldValue 把字段值编码成不含空格的形态（按 UTF-8 字节百分号转义）。
//
// 为什么渲染面必须转义而不是「解析面宽容地按多个空格切」：Kind/MimeType 是宿主
// 自由文本，含空格的形态真实存在（"image large"、带参数的 mime）。解析面按空格
// 切字段是格式契约，唯一能让两边一致的做法是渲染前把分隔符转义掉——否则往返
// 静默丢字段而 ok=true，错位的一份还会逐轮复用。
func escapeFieldValue(s string) string {
	need := false
	for i := 0; i < len(s); i++ {
		if !fieldSafe(s[i]) {
			need = true
			break
		}
	}
	if !need {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if fieldSafe(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(upperHex[c>>4])
		b.WriteByte(upperHex[c&0x0f])
	}
	return b.String()
}

// unescapeFieldValue 还原 escapeFieldValue 的编码。
//
// 宽容策略：不是合法 %XX 的 '%' 原样保留（历史占位文本里的字面 '%' 因此仍可读）。
// 新渲染的值里 '%' 恒被转义成 %25，故合法输入不会与旧文本混淆。
func unescapeFieldValue(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi, lo := hexVal(s[i+1]), hexVal(s[i+2])
			if hi >= 0 && lo >= 0 {
				b.WriteByte(byte(hi<<4 | lo))
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// hexVal 十六进制字符值；非十六进制返回 -1。
func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}
