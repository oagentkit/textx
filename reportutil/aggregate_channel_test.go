package reportutil

import "testing"

// finding 合成夹具：签名通道 + 判定等价的正文。
type finding struct {
	sig  string
	text string
}

// TestAggregateSharesChannelAcrossGroups 通道命中但 eq 不等价时，必须继续
// 试**共享该通道的其余簇**（textx#34）。
//
// 早先 byChannel 只存首见簇：A、B、C 共享通道 x 且 eq(A,B)=false、
// eq(B,C)=true 时，B 建的新簇拿不到通道注册，C 撞上 A 的簇、eq 不过便自成
// 第三簇——B/C 明明相互等价却永远分居两簇，Count 各为 1。
//
// 聚簇的用途正是「多份采样相互复现 ⇒ Count≥2 升级权重」，这种形态下复现
// 信号被静默拆散：三份采样看起来像三个各出现一次的问题。
func TestAggregateSharesChannelAcrossGroups(t *testing.T) {
	items := []finding{
		{sig: "x", text: "alpha"},
		{sig: "x", text: "beta"},
		{sig: "x", text: "beta"}, // 与第二项等价，与第一项不等价
	}
	groups := Aggregate(items,
		func(f finding) []string { return []string{f.sig} },
		func(a, b finding) bool { return a.text == b.text },
	)

	if len(groups) != 2 {
		t.Fatalf("应为 2 簇（alpha 一通、beta 两通），实得 %d 簇: %+v", len(groups), groups)
	}
	byText := map[string]int{}
	for _, g := range groups {
		byText[g.Representative.text] = g.Count
	}
	if byText["alpha"] != 1 {
		t.Errorf("alpha 的 Count=%d，应为 1", byText["alpha"])
	}
	if byText["beta"] != 2 {
		t.Errorf("beta 的 Count=%d，应为 2——复现信号被拆散（两簇各 1）", byText["beta"])
	}
}

// TestAggregateStillJoinsFirstMatch 成对对照：与首见簇等价时仍然并入它，
// 「继续试其余簇」不得把明确等价的项推去另建新簇。
func TestAggregateStillJoinsFirstMatch(t *testing.T) {
	items := []finding{
		{sig: "x", text: "same"},
		{sig: "x", text: "other"},
		{sig: "x", text: "same"},
	}
	groups := Aggregate(items,
		func(f finding) []string { return []string{f.sig} },
		func(a, b finding) bool { return a.text == b.text },
	)
	if len(groups) != 2 {
		t.Fatalf("应为 2 簇，实得 %d: %+v", len(groups), groups)
	}
	if groups[0].Representative.text != "same" || groups[0].Count != 2 {
		t.Errorf("首见簇应吸收等价项: %+v", groups[0])
	}
}

// TestAggregateMultiChannelLookup 多通道形态：一个簇在两个通道下都登记，
// 后到者无论走哪个通道都能被找到（而不是只有首见的那个通道知道它）。
func TestAggregateMultiChannelLookup(t *testing.T) {
	items := []finding{
		{sig: "xy", text: "alpha"}, // 通道 x 与 y 都登记
		{sig: "y", text: "beta"},   // 走 y 撞上 alpha，eq 不过 → 自成簇，登记在 y
		{sig: "y", text: "beta"},   // 走 y：同一通道下有两个簇，第二个等价
	}
	groups := Aggregate(items,
		func(f finding) []string {
			if f.sig == "xy" {
				return []string{"x", "y"}
			}
			return []string{f.sig}
		},
		func(a, b finding) bool { return a.text == b.text },
	)
	if len(groups) != 2 {
		t.Fatalf("应为 2 簇，实得 %d: %+v", len(groups), groups)
	}
	for _, g := range groups {
		if g.Representative.text == "beta" && g.Count != 2 {
			t.Fatalf("beta 的 Count=%d，应为 2", g.Count)
		}
	}
}

// TestAggregateDuplicateChannelInSignature 签名里重复的通道只登记一次：
// 否则同一个簇在同一通道下出现多份，匹配轮会重复试 eq（语义没错但白跑）。
func TestAggregateDuplicateChannelInSignature(t *testing.T) {
	items := []finding{
		{sig: "x", text: "alpha"},
		{sig: "x", text: "beta"},
		{sig: "x", text: "beta"},
	}
	groups := Aggregate(items,
		func(f finding) []string { return []string{f.sig, f.sig} }, // 故意重复
		func(a, b finding) bool { return a.text == b.text },
	)
	if len(groups) != 2 {
		t.Fatalf("应为 2 簇，实得 %d: %+v", len(groups), groups)
	}
	for _, g := range groups {
		if g.Representative.text == "beta" && g.Count != 2 {
			t.Fatalf("beta 的 Count=%d，应为 2", g.Count)
		}
	}
}
