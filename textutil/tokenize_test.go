package textutil

import (
	"reflect"
	"testing"
)

// 词元切分：固定测试向量（纯中文 / 中英混合 / 含标点 / 含引号与 FTS5 运算符）。
func TestTokens(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"空串", "", nil},
		{"纯中文逐字", "磁盘长期偏小", []string{"磁", "盘", "长", "期", "偏", "小"}},
		{"中英混合", "磁盘 usage 长期", []string{"磁", "盘", "usage", "长", "期"}},
		// 标点只切断词，不产出词元
		{"含标点", "Hello, world! 你好，世界。", []string{"Hello", "world", "你", "好", "世", "界"}},
		// 引号与 FTS5 运算符只做分隔，词元本身保留
		{"含引号与运算符", `foo "bar" NOT baz`, []string{"foo", "bar", "NOT", "baz"}},
		// FTS5 语法字符（* ( ) : ^ - ,）一律丢弃，不会漏进查询串
		{"含 FTS5 语法字符", "a*b (c) :d^ e-f", []string{"a", "b", "c", "d", "e", "f"}},
		// 大小写原样保留
		{"保留大小写", "Hello World", []string{"Hello", "World"}},
		// 数字算词的一部分
		{"数字", "v1.2.3 build 2024", []string{"v1", "2", "3", "build", "2024"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Tokens(c.in); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Tokens(%q) = %v，期望 %v", c.in, got, c.want)
			}
		})
	}
}

// 词元不去重：保留出现次数（TF 语义对调用方更有用）。
func TestTokensKeepsDuplicates(t *testing.T) {
	got := Tokens("磁盘磁盘")
	if !reflect.DeepEqual(got, []string{"磁", "盘", "磁", "盘"}) {
		t.Fatalf("Tokens 不应去重，得到 %v", got)
	}
}

// 索引用：入库前把文本切成空格分隔的词元串，
// unicode61 分词器才会把连续汉字切成一个一个单字词元。
func TestTokensJoined(t *testing.T) {
	if got := TokensJoined("磁盘长期偏小"); got != "磁 盘 长 期 偏 小" {
		t.Fatalf("TokensJoined = %q", got)
	}
	if got := TokensJoined(""); got != "" {
		t.Fatalf("空串应为空，得到 %q", got)
	}
}

// 查询串必须对 FTS5 语法安全：
//   - 词元加引号 → AND/OR/NOT/NEAR 降回字面量（不会被当成 NOT 条件）；
//   - 标点、运算符、引号不漏进查询串 → 任意用户输入都不构成语法错误。
func TestFTS5Query(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空串", "", ""},
		{"只有标点", "!!! ???", ""},
		{"纯中文", "磁盘", `"磁" "盘"`},
		{"中英混合", "磁盘 usage", `"磁" "盘" "usage"`},
		// NOT 是裸词运算符；加引号后它只是「查 NOT 这个词」
		{"NOT 不被当运算符", "foo NOT bar", `"foo" "NOT" "bar"`},
		{"引号被丢弃", `foo "bar"`, `"foo" "bar"`},
		{"FTS5 语法字符被丢弃", "a*b (c)", `"a" "b" "c"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FTS5Query(c.in); got != c.want {
				t.Fatalf("FTS5Query(%q) = %q，期望 %q", c.in, got, c.want)
			}
		})
	}
}

// 语义边界：**词的合取，不是连续子串**。
// "磁盘" 与 "盘磁" 切出同一组词元，顺序信息不保留——本函数分不出位置。
func TestTokensIsConjunctionNotSubstring(t *testing.T) {
	a := Tokens("磁盘")
	b := Tokens("盘磁")
	if len(a) != 2 || len(b) != 2 {
		t.Fatalf("词元数不对：%v / %v", a, b)
	}
	set := map[string]bool{}
	for _, tok := range a {
		set[tok] = true
	}
	for _, tok := range b {
		if !set[tok] {
			t.Fatalf("两者的词元集合应相同，得到 %v / %v", a, b)
		}
	}
	// 查询串保留书写顺序，但语义仍是「同时含」
	if got := FTS5Query("磁盘"); got != `"磁" "盘"` {
		t.Fatalf("查询串 = %q", got)
	}
}

// 与 BigramSet 并存、语义不同：二元组是「相邻两字」，词元是「单字」。
// 阈值按词元口径标定，换成二元组会把同一对文本的分数从 0.5 掉到 0.33。
func TestTokensVsBigramSet(t *testing.T) {
	s := "已修复"
	toks := Tokens(s)
	if !reflect.DeepEqual(toks, []string{"已", "修", "复"}) {
		t.Fatalf("Tokens = %v", toks)
	}
	bg := BigramSet(s)
	if !bg["已修"] || !bg["修复"] {
		t.Fatalf("BigramSet 口径漂移：%v", bg)
	}
}
