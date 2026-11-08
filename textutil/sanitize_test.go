// sanitize_test.go —— SanitizeFileStem 路径 token 消毒回归（textx#51）。
package textutil

import (
	"path/filepath"
	"testing"
)

// TestSanitizeFileStemDotTokens 消毒契约回归。既有映射面在 lines_test.go 的
// TestSanitizeFileStem；本表钉住 textx#51 的路径点 token 形态（"." / ".." 纯点
// 主干不含任何可映射字符，必须显式消解）与其余字符映射约定。
func TestSanitizeFileStemDotTokens(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"owner/repo", "owner-repo"}, // 路径分隔（GitLab 嵌套组）
		{"a#b", "a-b"},               // 引用语法
		{"a:b", "a-b"},               // 分支名
		{"a@b", "a-b"},
		{`a\b`, "a-b"},     // Windows 分隔
		{"../..", "..-.."}, // 含分隔符形态：'/' 映射后点不再构成穿越 token
		{"..", "-"},        // 纯 ".."：上级目录指代必须消解（textx#51）
		{".", "-"},         // 纯 "."：目录自身指代
		{"a..b", "a..b"},   // 内嵌点非路径 token，不动
		{"...", "..."},     // 非路径点 token（合法文件名主干），不动
		{"", ""},
	}
	for _, c := range cases {
		if got := SanitizeFileStem(c.in); got != c.want {
			t.Errorf("SanitizeFileStem(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSanitizeFileStemNoTraversalJoin 端到端语义：消毒结果作为路径段拼进
// 落档目录不得改变落点——纯 ".." 原样穿透时 filepath.Join 解析到父目录，
// 写入落在目标目录之外（doc 点名的目录穿越形态）。
func TestSanitizeFileStemNoTraversalJoin(t *testing.T) {
	joined := filepath.Join("/var/records", SanitizeFileStem(".."), "report.md")
	if !filepath.IsAbs(joined) || filepath.Dir(filepath.Dir(joined)) != "/var/records" {
		t.Fatalf("纯 \"..\" 消毒后拼路径仍逃出目标目录: %s", joined)
	}
}
