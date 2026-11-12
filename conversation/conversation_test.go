package conversation

import (
	"strings"
	"testing"
)

func mkTurns(n int) []Turn {
	turns := make([]Turn, 0, n)
	for i := 0; i < n; i++ {
		turns = append(turns, Turn{Question: "q", Answer: "a"})
	}
	return turns
}

func TestSplit(t *testing.T) {
	recent, evicted := Split(mkTurns(9), 6)
	if len(recent) != 6 || len(evicted) != 3 {
		t.Fatalf("9 轮 keep6 应切 6/3: %d/%d", len(recent), len(evicted))
	}
	recent, evicted = Split(mkTurns(6), 6)
	if len(recent) != 6 || evicted != nil {
		t.Fatalf("恰好 6 轮无淘汰: %d %d", len(recent), len(evicted))
	}
	recent, evicted = Split(mkTurns(3), 0)
	if len(recent) != 3 || evicted != nil {
		t.Fatalf("keep<=0 应全 verbatim: %d %d", len(recent), len(evicted))
	}
}

func TestRenderTruncates(t *testing.T) {
	long := strings.Repeat("长", 500)
	out := Render([]Turn{{Question: "q", Answer: long}})
	if strings.Count(out, "长") > AnswerCap+1 {
		t.Fatalf("回答应截断: %d", strings.Count(out, "长"))
	}
	if !strings.Contains(out, "用户: q") {
		t.Fatalf("问题应保留: %q", out)
	}
}

func TestCombine(t *testing.T) {
	out := Combine("早期结论 X", mkTurns(1))
	if !strings.Contains(out, "更早对话摘要") || !strings.Contains(out, "早期结论 X") {
		t.Fatalf("摘要应在头部: %q", out)
	}
	if out := Combine("", nil); out != "" {
		t.Fatalf("空会话应空注入: %q", out)
	}
}

// TestTruncateLineCapIncludesEllipsis（textx#88 B 面）：QuestionCap/AnswerCap 是
// 总长度预算——截断分支产出含省略号恒 ≤ cap（曾为 cap+1）；恰好 cap rune 的原文
// 不动（无内容被丢就不打省略号）；未触发截断时原样返回。
func TestTruncateLineCapIncludesEllipsis(t *testing.T) {
	for _, n := range []int{QuestionCap, AnswerCap} {
		got := truncateLine(strings.Repeat("长", n+100), n)
		if c := len([]rune(got)); c > n {
			t.Fatalf("cap=%d 截断产出 %d rune（省略号未计入上限，曾为 cap+1）", n, c)
		}
		if !strings.HasSuffix(got, "…") {
			t.Fatalf("cap=%d 超长应以省略号截断: %q", n, got)
		}
		if c := len([]rune(strings.TrimSuffix(got, "…"))); c != n-1 {
			t.Fatalf("cap=%d 截断内容应为 cap-1 rune: %d", n, c)
		}
	}
	// 恰好 cap：原文完整、不打省略号。
	if got := truncateLine(strings.Repeat("长", QuestionCap), QuestionCap); len([]rune(got)) != QuestionCap || strings.Contains(got, "…") {
		t.Fatalf("恰好 cap 不应截断: %q", got)
	}
	if got := truncateLine(strings.Repeat("长", 100), QuestionCap); len([]rune(got)) != 100 {
		t.Fatalf("未超 cap 应原样: %q", got)
	}
	// Render 级：单轮注入的问答内容行均 ≤ 对应 cap。
	out := Render([]Turn{{Question: "问" + strings.Repeat("问", 500), Answer: strings.Repeat("答", 500)}})
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "用户: "):
			if c := len([]rune(strings.TrimPrefix(line, "用户: "))); c > QuestionCap {
				t.Fatalf("问题行内容 %d > QuestionCap", c)
			}
		case strings.HasPrefix(line, "助手: "):
			if c := len([]rune(strings.TrimPrefix(line, "助手: "))); c > AnswerCap {
				t.Fatalf("回答行内容 %d > AnswerCap", c)
			}
		}
	}
}
