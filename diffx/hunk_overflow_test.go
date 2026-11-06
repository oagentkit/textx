package diffx

import (
	"math"
	"strconv"
	"testing"
)

// TestHunkLinesOverflowSkipped 行数/起始行越出 int 的 hunk 必须整块跳过
// （textx#32）。
//
// 旧实现 `cnt, _ = strconv.Atoi(m[2])` 把越界错误丢掉：Atoi 在溢出时返回
// `(MaxInt, ErrRange)`，随后 `start + cnt - 1` 整数下溢成 **end < start 的
// 负区间**。下游按闭区间取行会取错或取空——一段损坏（或恶意构造）的 diff
// 就能触发，且没有任何信号。
func TestHunkLinesOverflowSkipped(t *testing.T) {
	cases := []struct {
		name string
		diff string
	}{
		{
			name: "行数越界",
			diff: "+++ b/a.go\n@@ -1,1 +1,99999999999999999999 @@\n+x\n",
		},
		{
			name: "起始行越界",
			diff: "+++ b/a.go\n@@ -1,1 +99999999999999999999,5 @@\n+x\n",
		},
		{
			name: "加法溢出（两个加数各自合法）",
			diff: "+++ b/a.go\n@@ -1,1 +" + strconv.Itoa(math.MaxInt) + ",2 @@\n+x\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := HunkLines(c.diff)
			for file, ranges := range m {
				for _, r := range ranges {
					if r[1] < r[0] {
						t.Fatalf("%s: 区间下溢 %v（end < start）——下游按区间取行必错", file, r)
					}
					if r[0] < 0 || r[1] < 0 {
						t.Fatalf("%s: 区间出现负数 %v", file, r)
					}
				}
			}
			if len(m) != 0 {
				t.Fatalf("越界 hunk 应整块跳过，实得 %v", m)
			}
		})
	}
}

// TestHunkLinesNormalRangesStillParsed 成对对照：越界保护不得误伤合法区间
// （含恰好落在 int 边界上、不溢出的那种）。
func TestHunkLinesNormalRangesStillParsed(t *testing.T) {
	text := "+++ b/a.go\n@@ -1,3 +1,4 @@\n ctx\n+x\n+++ b/b.go\n@@ -1 +1 @@\n-x\n+y\n"
	m := HunkLines(text)
	if got := m["a.go"]; len(got) != 1 || got[0] != (LineRange{1, 4}) {
		t.Fatalf("a.go 区间错: %v", got)
	}
	if got := m["b.go"]; len(got) != 1 || got[0] != (LineRange{1, 1}) {
		t.Fatalf("b.go 区间错: %v", got)
	}

	// 边界但不溢出：start=MaxInt, cnt=1 ⇒ [MaxInt, MaxInt] 合法，不得被跳过。
	border := "+++ b/edge.go\n@@ -1,1 +" + strconv.Itoa(math.MaxInt) + ",1 @@\n+x\n"
	if got := HunkLines(border)["edge.go"]; len(got) != 1 || got[0] != (LineRange{math.MaxInt, math.MaxInt}) {
		t.Fatalf("边界单行区间应保留，实得 %v", got)
	}
}
