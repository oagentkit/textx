package textutil

import (
	"strings"
	"unicode"
)

// Tokens 把文本切成**检索用词元**。
//
// 切法：
//
//   - CJK 逐字成词元（"磁盘" → "磁", "盘"）——汉字/假名/谚文按单字切；
//   - 拉丁与数字按空白与标点切词（"disk is" → "disk", "is"），**原样保留大小写**；
//   - 只产出**裸词元**：标点、运算符、引号一律丢弃。
//
// 为什么 CJK 要逐字：SQLite FTS5 的 unicode61 分词器把连续汉字算作**一个词元**，
// 于是索引里的「磁盘长期偏小」是一个词元、查询「磁盘」是另一个词元，
// 两者永不相等 → **0 命中**，而且是**静默**返回空集（用户看到的是「库里没有这条」）。
// 真实中文文本没有空格分词，用户查的就是子串。
//
// 语义边界（重要，别当成通用分词器用）：
//
//   - 返回的是**词的合集**，不是连续子串。"磁盘" 与 "盘磁" 的词元集合相同，
//     但两者在原文里位置相反——本函数不保留位置信息，**分不出顺序**。
//   - **不支持短语检索**："disk is" 会被当成两个词的合取（都含），而不是相邻短语。
//     将来要支持短语，至少得能拿到词元 → 原文下标的位置信息。
//   - 与 BigramSet / Similarity **并存、不互相替代**：bigram 是字符二元组，
//     两者在**判定阈值**上不等价（中文「已修复」vs「未修复」：词元口径
//     Jaccard=0.5，二元组口径≈0.33）——阈值是按词元口径标定的，换口径会翻转判定。
//
// 词元**不去重**（保留出现次数，TF 语义对调用方更有用）。
func Tokens(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	var word []rune
	flush := func() {
		if len(word) > 0 {
			out = append(out, string(word))
			word = word[:0]
		}
	}
	for _, r := range s {
		switch {
		case isCJKIdeograph(r):
			flush()
			out = append(out, string(r))
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			word = append(word, r)
		default:
			// 标点、空白、运算符、引号：切断词，不产出词元。
			flush()
		}
	}
	flush()
	return out
}

// TokensJoined 把 Tokens 的结果用空格连接。
//
// **索引用**：FTS5 的 unicode61 分词器按空白切词，所以入库前把文本切成
// 空格分隔的词元串再存，索引里才会是「磁 盘 长 期」这样一个个单字词元。
// 查询侧用 FTS5Query（不是本函数——查询串要额外防运算符）。
func TokensJoined(s string) string {
	return strings.Join(Tokens(s), " ")
}

// FTS5Query 把文本切成词元并拼成 SQLite FTS5 安全的 MATCH 查询串。
//
// 每个词元加双引号后用空格连接：FTS5 的空格是隐式 AND，于是
// "磁" "盘" = 同时含「磁」与「盘」，正是中文子串检索要的效果。
//
// 加引号是**安全边界**，不是装饰：FTS5 的 AND / OR / NOT / NEAR 是**裸词**才生效的
// 运算符，用户输入里的 NOT 会被当成「排除后面那个词」而不是「查 NOT 这个词」；
// 引号把它降回字面量。引号同样能让 `"`、`*`、`(` 之类不构成语法错误。
//
// 空串（或切不出词元的串）返回空串——空 MATCH 是 FTS5 的语法错误，
// 调用方应据此跳过查询而不是拼上去。
func FTS5Query(s string) string {
	toks := Tokens(s)
	if len(toks) == 0 {
		return ""
	}
	parts := make([]string, len(toks))
	for i, t := range toks {
		// 词元里理论上不会有引号（切分时已丢弃），仍做转义：
		// 「不可信输入」这条路上不能留「理论上」。
		parts[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	return strings.Join(parts, " ")
}

// isCJKIdeograph 是否按「逐字成词元」处理的 CJK 表意文字。
//
// 判据刻意**窄于** runeUnitsCJK：后者把 CJK 标点与全角形式也算作宽字符
// （计量用），而切分时标点是要**丢弃**的分隔符，不该变成词元。
//
// 谚文（Hangul）也逐字：韩文以空格分词，但逐字切分退化成「含这些音节」的
// 合取检索——召回变宽，不会归零。
func isCJKIdeograph(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r)
}
