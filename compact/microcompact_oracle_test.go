package compact

// microcompact 的 token 口径必须**单一**（textx#33）。
//
// 失效形态是「两种口径相减」：TokensBefore 取 provider 真实用量、TokensSaved 是
// 本地估算的增量 ⇒ TokensAfter = before - saved 是个杂交量。它有两个可判定的后果：
//
//  1. 三个数自相矛盾（TokensAfter 既不等于真实剩余，也不等于任何本地估算）；
//  2. 更坏的是**门槛判错方向**：本地口径对中文系统性低估约 2 倍，真实能省几千
//     token 的清理在本地口径下只算几十，撞上 DefaultMinTokenSavings(256) 判
//     below_savings 不执行——真 token 已超窗，轻量先手却在等一个永远不会到来的
//     「够省」。
//
// 这组用例按「无 usage / 有 usage」**成对**写：单看有 usage 那条，一个把 saved
// 硬编码成大数的实现也能过；必须让同一份转写在本地口径下判 below_savings，
// 才能证明差异来自折算而不是来自别的什么。

import (
	"strings"
	"testing"
	"time"
)

// smallClearableTranscript 造一份「本地口径下节省不足门槛」的转写：
// 两组工具结果、KeepRecent=1 ⇒ 只清第一组；被清的那条约 52 token、
// 占位符 12 token ⇒ 本地节省 ≈ 40，低于 DefaultMinTokenSavings(256)。
func smallClearableTranscript() []Message {
	big := strings.Repeat("x", 156)  // 156 ASCII 单位 → 52 token
	small := strings.Repeat("y", 12) // 4 token（不被清：第二组是保留段）
	tc := ToolCall{ID: "a", Name: "read"}
	tc2 := ToolCall{ID: "b", Name: "read"}
	return []Message{
		{Role: RoleUser, Text: "u"},
		{Role: RoleAssistant, Text: "a", ToolCalls: []ToolCall{tc}},
		{Role: RoleTool, ToolCallID: "a", ToolName: "read", Text: big},
		{Role: RoleAssistant, Text: "a", ToolCalls: []ToolCall{tc2}},
		{Role: RoleTool, ToolCallID: "b", ToolName: "read", Text: small},
	}
}

// TestMicrocompactUsageScalesSavingsIntoProviderOracle 有 provider 用量时，
// 节省量必须折算到**同一个口径**再与门槛比、再与 before 相减（textx#33）。
func TestMicrocompactUsageScalesSavingsIntoProviderOracle(t *testing.T) {
	msgs := smallClearableTranscript()
	cfg := MicroConfig{KeepRecent: 1}
	now := time.Now()
	idle := now.Add(-2 * time.Hour) // idle 触发，与 token 阈值解耦

	localBefore := EstimateTokens(msgs)

	// 对照一：无 usage ⇒ 纯本地口径，节省 ~40 < 256 ⇒ below_savings。
	local := MaybeMicrocompact(msgs, cfg, idle, now)
	if local.Reason != MicroBelowSavings {
		t.Fatalf("前置：本地口径应判 below_savings，实得 %s", local.Reason)
	}
	// below_savings 路径不回报 TokensSaved（没清就没省），本地节省量自己算：
	// 被清条目的估值 − 占位符估值。
	localSaved := msgTokens(msgs[2]) - msgTokens(Message{Role: RoleTool, Text: ClearedPlaceholder})
	if localSaved <= 0 || localSaved > DefaultMinTokenSavings {
		t.Fatalf("前置：本地口径节省 %d 必须 >0 且不超过门槛 %d——"+
			"否则这条用例证不出折算方向", localSaved, DefaultMinTokenSavings)
	}
	if len(local.ClearedIDs) != 0 {
		t.Fatalf("below_savings 不得报 ClearedIDs（转写原样返回却报「清了这些」，"+
			"调用方按 id 去转写里找不到占位符）: %v", local.ClearedIDs)
	}

	// 对照二：provider 说这段转写实际用了 usage 倍于本地的 token ⇒ 同一份节省
	// 折算后远超门槛，必须执行。
	usage := localBefore * 100
	withUsage := MaybeMicrocompactWithUsage(msgs, cfg, idle, now, usage)
	if withUsage.Reason != MicroApplied {
		t.Fatalf("provider 口径下节省应按比例放大到门槛之上，实得 %s（saved=%d）——"+
			"门槛与 before 用不同口径比就是这里判错方向",
			withUsage.Reason, withUsage.TokensSaved)
	}
	if withUsage.TokensSaved <= DefaultMinTokenSavings {
		t.Fatalf("折算后的节省 %d 应远超门槛 %d（本地口径是 %d）",
			withUsage.TokensSaved, DefaultMinTokenSavings, localSaved)
	}
	// 折算比 = usage/localBefore ⇒ 节省 ≈ local.TokensSaved*100
	want := localSaved * 100
	if withUsage.TokensSaved != want {
		t.Fatalf("折算节省 = %d, want %d（本地 %d × 折算比 %d）——"+
			"折算比必须取 provider 用量 / 本地估算总量",
			withUsage.TokensSaved, want, localSaved, usage/localBefore)
	}
	// 三个数必须自洽：before - saved == after。
	if withUsage.TokensBefore != usage {
		t.Fatalf("TokensBefore = %d, want provider 用量 %d", withUsage.TokensBefore, usage)
	}
	if withUsage.TokensBefore-withUsage.TokensSaved != withUsage.TokensAfter {
		t.Fatalf("口径不自洽：before(%d) - saved(%d) != after(%d)——三个数跨口径即为杂交量",
			withUsage.TokensBefore, withUsage.TokensSaved, withUsage.TokensAfter)
	}
}

// TestMicrocompactNoUsageKeepsLocalOracle 成对对照：无 provider 用量时三个数
// 全走本地口径，不得因上面那条引入折算而漂移。
func TestMicrocompactNoUsageKeepsLocalOracle(t *testing.T) {
	msgs := mkTranscript(6, 1) // 每条工具结果 3000 字符 → 1000 token，远超门槛
	cfg := MicroConfig{KeepRecent: 1}
	now := time.Now()
	localBefore := EstimateTokens(msgs)

	r := MaybeMicrocompact(msgs, cfg, now.Add(-2*time.Hour), now)
	if r.Reason != MicroApplied {
		t.Fatalf("应应用，实得 %s", r.Reason)
	}
	if r.TokensBefore != localBefore {
		t.Fatalf("无 usage 时 TokensBefore 必须是本地估算 %d，实得 %d", localBefore, r.TokensBefore)
	}
	if r.TokensBefore-r.TokensSaved != r.TokensAfter {
		t.Fatalf("本地口径也应自洽：%d - %d != %d", r.TokensBefore, r.TokensSaved, r.TokensAfter)
	}
	// 本地口径下节省应等于「估值差」，不经过折算
	if r.TokensSaved != localBefore-EstimateTokens(r.Messages) {
		t.Fatalf("saved = %d, 转写前后差 = %d（无 usage 时两者必须相等）",
			r.TokensSaved, localBefore-EstimateTokens(r.Messages))
	}
}
