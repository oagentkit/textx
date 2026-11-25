package spill

// issue #199 的判据：分页取回对多字节字符无损。
//
// Page 的字段全带 json 标签、直接进 read_spill 回包/控制台取回面——页边界
// 切坏字符产出的非法 UTF-8 会被 encoding/json 替换为 U+FFFD，每个跨页字符
// 静默丢 1–2 字节且不可复原。这正与截断提示的承诺（「全文已存档…分页取回
// 中段原文」）冲突：截断无损，取回却有损。
//
// 验收（issue 原文）：
//   - 正文「内容甲乙丙」、limit=4 ⇒ 每页 utf8.ValidString(Content) 为真，
//     且逐页拼接后与原文相等；
//   - limit=1（小于字符宽度）不死循环，且同样无损。
import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func writeSpill(t *testing.T, body string) (root, name string) {
	t.Helper()
	root = t.TempDir()
	dir := filepath.Join(root, "s1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	name = "u-spill.txt"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, name
}

// pageAll 沿 NextOffset 顺序取回整份存档（分页器的真实用法），带死循环护栏。
func pageAll(t *testing.T, root, name string, limit int) ([]Page, string) {
	t.Helper()
	var pages []Page
	var b strings.Builder
	for off := int64(0); ; {
		pg, err := ReadPage(root, "s1", name, off, limit)
		if err != nil {
			t.Fatalf("分页取回失败（off=%d limit=%d）: %v", off, limit, err)
		}
		if !utf8.ValidString(pg.Content) {
			t.Fatalf("页边界切出了非法 UTF-8（off=%d limit=%d content=%q）", off, limit, pg.Content)
		}
		b.WriteString(pg.Content)
		pages = append(pages, pg)
		if pg.EOF {
			return pages, b.String()
		}
		if pg.NextOffset <= off {
			t.Fatalf("NextOffset 不前进（off=%d → %d）：分页器死循环", off, pg.NextOffset)
		}
		off = pg.NextOffset
		if int64(len(pages)) > int64(len(b.String()))+8 { // 死循环护栏（每页至少 1 字节前进）
			t.Fatal("分页迟迟不到 EOF（疑似死循环）")
		}
	}
}

// TestReadPageUTF8SurvivesPageBoundary 验收一：中文正文、limit=4（预算终点
// 落在 3 字节字符中间）——每页合法 UTF-8，逐页拼接与原文相等。
func TestReadPageUTF8SurvivesPageBoundary(t *testing.T) {
	const body = "内容甲乙丙" // 5 汉字 × 3 字节 = 15 字节；limit=4 必切进字符
	root, name := writeSpill(t, body)

	pages, joined := pageAll(t, root, name, 4)
	if joined != body {
		t.Fatalf("逐页拼接必须与原文逐字节相等（实得 %q）", joined)
	}
	if len(pages) < 4 {
		t.Fatalf("limit=4、15 字节应至少 4 页（实得 %d）", len(pages))
	}
	for _, pg := range pages {
		if int(pg.Bytes) > len(pg.Content) {
			t.Fatalf("Bytes 应与 Content 字节数一致: %+v", pg)
		}
		if pg.Bytes > 4+utf8.UTFMax { // 进度保底的超额至多 UTFMax-1
			t.Fatalf("单页超额不得超出一个字符宽度: %+v", pg)
		}
	}
}

// TestReadPageLimitSmallerThanRuneWidth 验收二：limit=1（小于单字符宽度）
// 不死循环，每页一个完整字符，拼接无损。
func TestReadPageLimitSmallerThanRuneWidth(t *testing.T) {
	const body = "内容甲乙丙"
	root, name := writeSpill(t, body)

	pages, joined := pageAll(t, root, name, 1)
	if joined != body {
		t.Fatalf("limit=1 仍须无损（实得 %q）", joined)
	}
	if len(pages) != 5 {
		t.Fatalf("每页应恰好一个字符（实得 %d 页）", len(pages))
	}
	for i, pg := range pages {
		if utf8.RuneCountInString(pg.Content) != 1 {
			t.Fatalf("第 %d 页应含恰好一个完整字符: %q", i, pg.Content)
		}
	}
}

// TestReadPageASCIIPagingUnchanged ASCII 正文分页语义与修复前逐条一致
// （既有 TestReadPagePaging 的口径在 rune 安全化后不得漂移）。
func TestReadPageASCIIPagingUnchanged(t *testing.T) {
	root, name := writeSpill(t, strings.Repeat("a", 100))

	p1, err := ReadPage(root, "s1", name, 0, 40)
	if err != nil || p1.Bytes != 40 || p1.NextOffset != 40 || p1.EOF {
		t.Fatalf("首页形态不符: %+v err=%v", p1, err)
	}
	p3, err := ReadPage(root, "s1", name, 80, 40)
	if err != nil || p3.Bytes != 20 || !p3.EOF || p3.NextOffset != 0 {
		t.Fatalf("尾页形态不符: %+v err=%v", p3, err)
	}
	// 越界 offset 收到 total：空尾页 EOF。
	pg, err := ReadPage(root, "s1", name, 999, 10)
	if err != nil || pg.Offset != 100 || pg.Bytes != 0 || !pg.EOF {
		t.Fatalf("越界 offset 钳制不符: %+v err=%v", pg, err)
	}
}

// TestReadPageMidRuneOffsetRealigns 头部回退：非顺序分页传入落在字符中间
// 的 off——页重叠而非跳字节，Content 仍从字符起点起、合法 UTF-8，
// 与顺序分页拼接的结果一致。
func TestReadPageMidRuneOffsetRealigns(t *testing.T) {
	const body = "内容甲乙丙"
	root, name := writeSpill(t, body)

	// off=1 落在「内」的第二字节：应回退到 0。
	pg, err := ReadPage(root, "s1", name, 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	if pg.Offset != 0 {
		t.Fatalf("落在字符中间的 off 应回退到字符起点（实得 offset=%d）", pg.Offset)
	}
	if !utf8.ValidString(pg.Content) || !strings.HasPrefix(body, pg.Content) {
		t.Fatalf("回退后应从字符起点起（实得 %q）", pg.Content)
	}
	// off=10 落在「乙」（9..12）的中间：应回退到 9（乙的起点）并取回整字符。
	pg, err = ReadPage(root, "s1", name, 10, 4)
	if err != nil {
		t.Fatal(err)
	}
	if pg.Offset != 9 || pg.Content != "乙" {
		t.Fatalf("off=10 应回退到 9 并取回「乙」（实得 offset=%d content=%q）", pg.Offset, pg.Content)
	}
}

// TestReadPageRuneAlignedNextOffset 钉 NextOffset 恒为字符起点：顺序分页的
// 下一个 off 直接可用（这正是「尾部回退后 NextOffset 用回退后的量」的意义
// ——跨页字符整字符挪到下一页，无丢无损）。
func TestReadPageRuneAlignedNextOffset(t *testing.T) {
	const body = "甲乙丙丁"
	root, name := writeSpill(t, body)
	off := int64(0)
	for off < int64(len(body)) {
		pg, err := ReadPage(root, "s1", name, off, 5)
		if err != nil {
			t.Fatal(err)
		}
		if pg.EOF {
			break
		}
		if !utf8.ValidString(pg.Content) {
			t.Fatalf("页 %d 内容非法 UTF-8: %q", off, pg.Content)
		}
		// NextOffset 必须落在字符边界（正文是纯 3 字节字符：offset 必为 3 的倍数）。
		if pg.NextOffset%3 != 0 {
			t.Fatalf("NextOffset 应落在字符起点（实得 %d）", pg.NextOffset)
		}
		off = pg.NextOffset
	}
}
