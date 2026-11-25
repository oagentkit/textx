package budget

import (
	"context"
	"testing"
)

// TestUsageApplySlicingIsExecutable docs/USAGE.md 的官方接线示例夹具：
// Items=10、KeepTail=4 时必须无错，且产出 5 段（1 摘要 + 4 逐字尾部）。
//
// 这一条专门钉住**切片下标**：尾部从 plan.Cutoff 起切，不是从
// plan.KeepTailItems 起切——后者是「保留几条」，不是「从哪里开始」。
// 两者在 KeepTail==4 时数值恰好等于 Items-Cutoff，容易抄错。
func TestUsageApplySlicingIsExecutable(t *testing.T) {
	items := items(10, 25, "msg")
	plan, err := BuildPlan(PlanInput{Items: len(items), KeepTail: 4})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	res, err := Apply(context.Background(), plan,
		items[:plan.Cutoff], items[plan.Cutoff:], // 与 docs/USAGE.md 的官方示例逐字一致
		&fakeArchive{}, &fakeSummarizer{summary: Summary{Text: "摘要", Tokens: 40}})
	if err != nil {
		t.Fatalf("文档示例应可跑通：%v", err)
	}
	if len(res.Segment) != 5 {
		t.Fatalf("压实后应产出 5 段（1 摘要 + 4 尾部），实得 %d", len(res.Segment))
	}
}
