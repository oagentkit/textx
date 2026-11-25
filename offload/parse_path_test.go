package offload

// 占位文本解析面的 path 校验（issue #283）。
//
// 解析面是**读入**面：文本来自模型历史、模型复述或被注入的外部内容。写入侧的
// StorePath 净化只对写入侧成立；解析面原样收下 path，宿主按它取回时 `..` 与
// 分隔符就变成路径穿越。两个面必须共用同一份逻辑键判据。

import (
	"strings"
	"testing"
)

// TestParseRejectsUnsafePath 不可信 path（穿越/绝对/空白/反斜杠）一律判失败。
func TestParseRejectsUnsafePath(t *testing.T) {
	cases := []string{
		"../../etc/passwd",
		"blobs/ab/../../etc/passwd",
		"/etc/passwd",
		"blobs//ab/x.png",
		"./x",
		"ab cd",
		`blobs\ab\x.png`,
		"blobs/ab/x..png",
		"日",
	}
	for _, p := range cases {
		text := "[外置对象 kind=image id=abcdef path=" + p + " bytes=1]"
		if got, ok := Parse(text); ok {
			t.Fatalf("path %q 应被拒，实得 %+v", p, got)
		}
	}
}

// TestParseAcceptsStorePathOutput 写入侧产出的键**一定**能被解析面接受。
func TestParseAcceptsStorePathOutput(t *testing.T) {
	for _, h := range []string{"ab", "abcdef0123456789", strings.Repeat("0f", 32)} {
		for _, ext := range []string{".png", "bin", ".tar.gz", "A_b-9"} {
			p, err := StorePath(h, ext)
			if err != nil {
				t.Fatalf("StorePath(%q,%q): %v", h, ext, err)
			}
			text := Render(Ref{Kind: "image", ID: TruncatedHash(h), Path: p, Bytes: 4096}, "")
			got, ok := Parse(text)
			if !ok || got.Path != p {
				t.Fatalf("写入侧键 %q 应可被解析面接受，实得 %+v ok=%v", p, got, ok)
			}
		}
	}
}

// TestParseAcceptsSafeLogicalKey 不含穿越形态的相对逻辑键仍可解析（不回退）。
func TestParseAcceptsSafeLogicalKey(t *testing.T) {
	for _, p := range []string{"p", "blobs/ab/x.png", "a/b/c"} {
		got, ok := Parse("[外置对象 kind=image id=abcdef path=" + p + " bytes=1]")
		if !ok || got.Path != p {
			t.Fatalf("安全逻辑键 %q 应可解析，实得 %+v ok=%v", p, got, ok)
		}
	}
}
