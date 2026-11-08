// sanitize.go —— 落盘文件名安全化（textutil 通用文本/标识处理）。
package textutil

import "strings"

// SanitizeFileStem 文件名安全化：标识串（owner/repo/number 等外部标识）里的
// 路径分隔与引用语法字符归一为 "-"。外部标识常含 "/"（GitLab 嵌套组）、"#"、
// ":"（分支名）、甚至 ".."——未消毒直接拼进留档文件名会导致写入失败甚至
// 目录穿越。只处理文件名主干（不含扩展名），调用方自行拼接后缀。
//
// 纯路径点主干（"." / ".."）不含任何可映射字符、按字符映射会原样穿透，而它
// 们正是目录指代 token：拼进路径段即解析到目录自身/上级目录——doc 点名的
// 穿越威胁的最简形态（textx#51）。归一为 "-"，与其余被消毒字符同待遇；
// 内嵌点（"a..b"、"..."）非路径 token，不动。
func SanitizeFileStem(s string) string {
	if s == "." || s == ".." {
		return "-"
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '#', '@', ':', '\\':
			return '-'
		}
		return r
	}, s)
}
