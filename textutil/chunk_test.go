package textutil

import (
	"strings"
	"testing"
)

func TestChunkLinesPacksToTarget(t *testing.T) {
	md := "第一行\n第二行\n第三行\n第四行\n"
	chunks := ChunkLines(md, 12, 400)
	if len(chunks) == 0 {
		t.Fatal("应产出分片")
	}
	// join 严格还原
	if got := strings.Join(chunks, ""); got != md {
		t.Fatalf("join 应严格等于原文，得到 %q", got)
	}
	// 每片不超过 target（rune）
	for i, c := range chunks {
		if n := len([]rune(c)); n > 12 {
			t.Fatalf("片 %d 超过 target：%d rune", i, n)
		}
	}
}

func TestChunkLinesHardSplitsLongLine(t *testing.T) {
	long := strings.Repeat("字", 500)
	chunks := ChunkLines(long, 160, 400)
	total := ""
	for _, c := range chunks {
		total += c
		if len([]rune(c)) > 400 {
			t.Fatalf("硬切上限失效：片长 %d", len([]rune(c)))
		}
	}
	if total != long {
		t.Fatal("硬切后 join 仍应还原全文")
	}
	// 硬切片独立成片（flush 后不入后续凑片缓冲）
	if chunks[0] != strings.Repeat("字", 400) {
		t.Fatalf("首片应为 400 rune 硬切，实际 %d", len([]rune(chunks[0])))
	}
}

func TestChunkLinesKeepsTableRowsIntact(t *testing.T) {
	row := "| 列1 | 列2 | 列3 |\n"
	chunks := ChunkLines(strings.Repeat(row, 3), 45, 400)
	for _, c := range chunks {
		if strings.Contains(c, "\n") && !strings.HasSuffix(c, "\n") {
			t.Fatalf("片不应在行中间断裂：%q", c)
		}
	}
}

func TestChunkLinesEmpty(t *testing.T) {
	if got := ChunkLines("   \n  ", 160, 400); got != nil {
		t.Fatalf("空白文本应返回 nil，得到 %v", got)
	}
}

func TestChunkLinesDefaults(t *testing.T) {
	// target/hard<=0 走缺省（160/400）：500 rune 单行被 400 硬切
	chunks := ChunkLines(strings.Repeat("字", 500), 0, 0)
	if len([]rune(chunks[0])) != 400 {
		t.Fatalf("缺省 hard=400 未生效：%d", len([]rune(chunks[0])))
	}
}

func TestClampRuneBoundary(t *testing.T) {
	s := "中文abc" // 6+3=9 字节：中文各 3 字节
	if got := ClampRuneBoundary(s, 4); got != 3 {
		t.Fatalf("切进中文字符应回退到边界，得到 %d", got)
	}
	if got := ClampRuneBoundary(s, 3); got != 3 {
		t.Fatalf("边界处不动，得到 %d", got)
	}
	if got := ClampRuneBoundary(s, 100); got != len(s) {
		t.Fatalf("超长应钳到 len(s)，得到 %d", got)
	}
	if got := ClampRuneBoundary(s, 0); got != 0 {
		t.Fatalf("0 应返回 0，得到 %d", got)
	}
}

func TestEstTokensCJK(t *testing.T) {
	if got := EstTokensCJK(""); got != 0 {
		t.Fatalf("空串应为 0，得到 %d", got)
	}
	// 纯中文：8 字 ×2 单位 = 16 units → 16/3+1 = 6（纯 rune 口径只给 3——中文低估的修正点）
	if got := EstTokensCJK("诊断结论完整覆盖"); got != 6 {
		t.Fatalf("中文估算漂移，得到 %d", got)
	}
	// 纯 ASCII：13 runes → 13/3+1 = 5（两口径一致）
	if got := EstTokensCJK("hello world!!"); got != 5 {
		t.Fatalf("英文估算漂移，得到 %d", got)
	}
	// 全角标点按 2 计
	if got := EstTokensCJK("，："); got != 2 { // 4 units/3+1 = 2
		t.Fatalf("全角标点应按 2 计，得到 %d", got)
	}
}

// TestChunkLinesRealBoundMaxTargetHard（textx#87）：三方承诺（godoc/README/测试）
// 的真实上界钉源——单行（含行尾换行）落进 (target, hard] 时整行成片、片长可到
// hard；超 hard 在 hard 硬切；全短行输入上界收紧回 target。join 恒还原原文。
// （旧夹具每行仅 3 rune，绕开了触发区，契约断言不具备把关能力。）
func TestChunkLinesRealBoundMaxTargetHard(t *testing.T) {
	const target, hard = 160, 400
	bound := max(target, hard)
	mk := map[string]string{
		"单行=target含换行(target+1)":  strings.Repeat("字", target) + "\n",
		"单行300rune∈(target,hard]": strings.Repeat("字", 300) + "\n",
		"单行=hard":                 strings.Repeat("字", hard) + "\n",
		"单行>hard":                 strings.Repeat("字", hard+1),
		"短行夹长行混合":                 strings.Repeat("短行\n", 5) + strings.Repeat("长", 300) + "\n" + strings.Repeat("短行\n", 5),
	}
	for name, src := range mk {
		chunks := ChunkLines(src, target, hard)
		if got := strings.Join(chunks, ""); got != src {
			t.Fatalf("%s: join 应严格等于原文", name)
		}
		for i, c := range chunks {
			if n := len([]rune(c)); n > bound {
				t.Fatalf("%s: 片 %d 超 max(target,hard)=%d：%d rune", name, i, bound, n)
			}
		}
	}

	// 对照组：全部单行（含换行）≤ target ⇒ 凑片路径上界收紧回 target。
	short := strings.Repeat("短内容行\n", 40)
	for i, c := range ChunkLines(short, target, hard) {
		if n := len([]rune(c)); n > target {
			t.Fatalf("全短行片 %d 超 target：%d rune", i, n)
		}
	}

	// 消费方要严格上界 R：传 target=hard=R 两档合一。
	strict := strings.Repeat("字", hard+57)
	for i, c := range ChunkLines(strict, bound, bound) {
		if n := len([]rune(c)); n > bound {
			t.Fatalf("target=hard 严格上界失效，片 %d：%d rune", i, n)
		}
	}
}
