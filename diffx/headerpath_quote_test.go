package diffx

import (
	"strings"
	"testing"
)

// TestHeaderPathGitQuotedForm 真实 git 的 quoted 形态（textx#35）：
// git 的 quote_two 语义把**侧前缀与路径整体**加引号，所以剥前缀必须在
// unquote **之后**——反过来（v0.12.0 的实现顺序：先 TrimPrefix("b/") 再查引号）
// 对真实输出永不命中（首字符是 `"`），`b/` 会跟着 unquote 一起留在结果里。
//
// 这不是边角情况：git 出厂 core.quotepath=true，**任何非 ASCII 文件名**都会
// 输出成 `+++ "b/\346\226\207…"` 形态。残留 b/ ⇒ 下游按真实仓库路径查 map
// 必落空 ⇒ 该文件的 diff 静默丢失（argus 的 fillFromRawDiff 实测形态）。
func TestHeaderPathGitQuotedForm(t *testing.T) {
	const (
		plus  = "+++ "
		minus = "--- "
	)
	// 中文「文件」的 UTF-8 字节 E6 96 87 E4 BB B6 的八进制转义。
	// 用反引号字符串让 `\346` 保持「反斜杠 + 三位数字」的字面形态，
	// 这正是 git 输出的样子（不是 Go 的转义）。
	cases := []struct {
		name, line, prefix, want string
	}{
		{"引号内含侧前缀", `+++ "b/a\"b.go"`, plus, `a"b.go`},
		{"非 ASCII 八进制转义", `+++ "b/\346\226\207\344\273\266.go"`, plus, "文件.go"},
		{"--- 侧 a/ 前缀", `--- "a/\346\226\207\344\273\266.go"`, minus, "文件.go"},
		{"引号 + 行尾附加信息", `+++ "b/\346\226\207\344\273\266.go"` + "\t2026-01-02", plus, "文件.go"},
		{"引号 + CRLF", `+++ "b/a b.go"` + "\r", plus, "a b.go"},
		// 既有行为不得回退：前缀在引号外的形态（git 不产生，但 v0.12.0 的实现
		// 只覆盖了它——两种形态都要支持，故两处各判一次而不是调换顺序）。
		{"前缀在引号外（既有形态）", `+++ b/"a b.go"`, plus, "a b.go"},
		{"裸路径 + b/ 前缀", `+++ b/x.go`, plus, "x.go"},
		{"--no-prefix 裸路径", `+++ x.go`, plus, "x.go"},
		{"quoted 且无侧前缀", `+++ "\346\226\207\344\273\266.go"`, plus, "文件.go"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := headerPath(c.line, c.prefix)
			if !ok {
				t.Fatalf("headerPath(%q) 应识别为头行", c.line)
			}
			if got != c.want {
				t.Fatalf("headerPath(%q) = %q，want %q", c.line, got, c.want)
			}
			// 机制断言：结果里不得残留侧前缀——这正是旧实现在真实 git 输出上的
			// 形态（btw 「路径恰好以 b/ 开头」的真文件名不在此列，故只对上面这些
			// 不以 b/ 开头的期望值判）。
			if !strings.HasPrefix(c.want, "b/") && strings.HasPrefix(got, "b/") {
				t.Fatalf("路径残留侧前缀 b/：%q", got)
			}
		})
	}
}

// TestDiffPathGitEscapedPath 端到端：真实 git 对非 ASCII 文件名输出的整块
// diff，DiffPath 必须还原成仓库里的真实路径。
func TestDiffPathGitEscapedPath(t *testing.T) {
	chunk := "diff --git a/\346\226\207\344\273\266.go b/\346\226\207\344\273\266.go\n" +
		"+++ \"b/\\346\\226\\207\\344\\273\\266.go\"\n" + // 注意：这里是 git 的字面转义形态
		"@@ -1 +1 @@\n-a\n+b\n"
	if got := DiffPath(chunk); got != "文件.go" {
		t.Fatalf("DiffPath = %q，want 文件.go（残留 b/ 会让下游按真实路径查 map 落空）", got)
	}
	// 删除文件形态：+++ /dev/null，回退 --- 侧的 quoted 路径
	chunk2 := "diff --git a/\346\226\207\344\273\266.go b/\346\226\207\344\273\266.go\n" +
		"--- \"a/\\346\\226\\207\\344\\273\\266.go\"\n+++ /dev/null\n"
	if got := DiffPath(chunk2); got != "文件.go" {
		t.Fatalf("删除文件回退 --- 侧 = %q，want 文件.go", got)
	}
}
