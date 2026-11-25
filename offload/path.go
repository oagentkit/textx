package offload

// 逻辑键推导与形状判据：写入侧（StorePath 构造）与解析面（Parse 校验）共用。
//
// 单独成文件是为了让「同一份判据」的边界一眼可见：sanitizeExt / validRelPath /
// isHexString 三处是键形状的全部规则，任何一侧都不得另写一套。

import (
	"strings"
)

// sanitizeExt 扩展名净化：补前导点、限字符集、拒分隔符与纯点串。
// 返回带点形态（如 ".png"），便于直接拼键。
func sanitizeExt(ext string) (string, error) {
	e := strings.TrimSpace(ext)
	e = strings.TrimPrefix(e, ".")
	if e == "" || strings.Trim(e, ".") == "" || strings.HasPrefix(e, ".") || strings.Contains(e, "..") {
		return "", ErrBadExt
	}
	if strings.ContainsAny(e, `/\`) {
		return "", ErrBadExt
	}
	for _, r := range e {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return "", ErrBadExt
		}
	}
	return "." + e, nil
}

// validRelPath 逻辑键的形状判据：**写入侧与解析面共用这一份**。
//
// 为什么必须共用：写入侧的 StorePath 净化只保证「我们产出的键安全」，而解析面
// 读的是模型历史里的文本，可来自复述或被注入的外部内容。两边各写一套判据，
// 漂移是必然的；这里只写一份，StorePath 产出后自检、Parse 收下前校验。
//
// 判据：相对路径（不以 / 开头）、段非空、无 "." / ".." 段、段内不含 ".."、
// 字符集限于字母数字与 . _ -。这排除了路径穿越、绝对路径、反斜杠、空白与
// 非 ASCII（StorePath 的键只由十六进制摘要与净化扩展名组成，必然满足）。
func validRelPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || strings.Contains(seg, "..") {
			return false
		}
		for _, r := range seg {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			case r == '.', r == '_', r == '-':
			default:
				return false
			}
		}
	}
	return true
}

// StorePath 由内容摘要与扩展名推导逻辑落盘键：blobs/<前两位>/<摘要><扩展名>。
//
// 纯函数，且**同一 (hash, ext) 恒得同一键**——这是去重的全部机制：键一致，
// 占位文本才可能逐字节一致，历史前缀才能命中提示缓存。按摘要前两位分片只是
// 避免单目录堆积，宿主可忽略目录层级自行映射。
//
// 返回的键以 path 分隔（与宿主无关的逻辑键，不是本机文件路径）。
func StorePath(hash, ext string) (string, error) {
	h := strings.TrimSpace(hash)
	if !isHexString(h) {
		return "", ErrBadHash
	}
	e, err := sanitizeExt(ext)
	if err != nil {
		return "", err
	}
	rel := "blobs/" + h[:2] + "/" + h + e
	if !validRelPath(rel) {
		// 组件都已净化时不可达；保留自检是为了让「写入侧与解析面同一判据」
		// 成为机制而不是注释里的承诺。
		return "", ErrBadHash
	}
	return rel, nil
}

// isHexString 报告 s 是否为至少两位的十六进制串。
//
// 为什么必须严判字符集而不是只查长度：h[:2] 会直接拼进分片目录，`/`、`..` 与
// 任意字节让键变成路径穿越的载体；多字节 rune 还会被劈成非法 UTF-8 目录名。
// 长度下限与字符集同属 ErrBadHash 的文案契约（「至少 2 位十六进制」）。
func isHexString(s string) bool {
	if len(s) < 2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
