package offload

// 落盘键推导的摘要字符集判据（issue #282）。
//
// ErrBadHash 的文案承诺「至少 2 位十六进制」，而实现只查长度：键里因此可以出现
// `/`、空格、任意字节，`h[:2]` 还能把多字节 rune 劈成半个分片目录。StorePath 是
// 导出 API，宿主常把键拼进目录——键里的分隔符与 `..` 直接变成路径穿越。

import (
	"strings"
	"testing"

	"github.com/oekit/pkg/errors"
)

// TestStorePathRejectsNonHexHash 非十六进制摘要一律拒绝：契约文案是判据的一部分。
func TestStorePathRejectsNonHexHash(t *testing.T) {
	cases := []string{
		"ab/../../etc/passwd",
		"ab cd",
		"日",
		"ab-cd",
		"ab_cd",
		"ab.cd",
		"zz",
		"ab\x00cd",
	}
	for _, h := range cases {
		if got, err := StorePath(h, ".png"); err == nil {
			t.Fatalf("摘要 %q 应被拒，实得键 %q", h, got)
		} else if !errors.Is(err, ErrBadHash) {
			t.Fatalf("摘要 %q 应返回 ErrBadHash，实得 %v", h, err)
		}
	}
}

// TestStorePathAcceptsHexHash 正常十六进制摘要不回退（长度 >= 2，大小写均可）。
func TestStorePathAcceptsHexHash(t *testing.T) {
	for _, h := range []string{"abc", "ABCDEF", strings.Repeat("0f", 32)} {
		p, err := StorePath(h, ".png")
		if err != nil {
			t.Fatalf("十六进制摘要 %q 应合法: %v", h, err)
		}
		if !strings.HasPrefix(p, "blobs/"+h[:2]+"/") || !strings.HasSuffix(p, ".png") {
			t.Fatalf("键形态不符: %q", p)
		}
	}
}

// TestStorePathShardStaysValidUTF8 分片前缀必须来自合法十六进制，不得劈开 rune。
func TestStorePathShardStaysValidUTF8(t *testing.T) {
	if _, err := StorePath("日", ".png"); err == nil {
		t.Fatal("非 ASCII 摘要不得产出分片目录名")
	}
}
