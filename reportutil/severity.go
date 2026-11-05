// Package reportutil 评审/评测报告后处理原语（沉淀自 heimdallr 报告层与
// bianque 评审流水线），一个包覆盖同一条消费链的三段：
//
//   - 严重度归一（severity.go）：critical/high/medium/low 四档词表 + 外部
//     审查专家常用别名（P0/fatal/nit…）折叠——「严重级别是什么、多严重」；
//   - 采样聚簇（cluster.go）：best-of-N 输出按签名通道聚簇计数（含
//     file+规范化文本精确指纹），「哪些发现被独立复现」；
//   - 对比统计（stats.go）：小样本通过率 Wilson 区间 + 配对 McNemar 精确
//     检验——「两个版本的差异是否显著」。
//
// 三段原为 severity/sampling/stats 三包，消费者画像相同且边界已漂移过一轮
// （v0.10.9 severity→sampling 指纹迁移），v0.10.11 合并为单包。零外部依赖。
package reportutil

import "strings"

// 严重级别词表（四档：critical > high > medium > low）。
//
// ⚠️ v0.8.0 起为四档（oagentkit/textx#20）：此前 critical 被折进 high、刻度只有
// 3/2/1，四档宿主（审批闸按档位裁决）被迫在消费侧保留 critical 特判残根，
// 忘了特判就与 high 静默同档、顶格语义丢失。Rank 刻度随之改 4/3/2/1——
// high/medium/low 的相对序不变，仅顶格档从 high 之上多出一级。
const (
	Critical = "critical"
	High     = "high"
	Medium   = "medium"
	Low      = "low"
)

// aliases 外部审查专家的常见严重度词表：cli/acpx 类外部 agent 常用 P0/P1/
// nit 等通用研发词表而非 high/medium/low，缺别名会让外部专家的 P0 整批落
// low（高危意见永远触发不了高严重度裁决）。
//
// P0-P3 按 4 档刻度取自然映射（P0→critical / P1→high / P2→medium / P3→low）
// ——四档化后若仍把 P1 折进 medium、P0 与 P1 之间会空一档，别名表自身先
// 不自洽。fatal/blocker/致命/阻断 = 顶格语义（crash/release blocker）。
var aliases = map[string]string{
	"p0":      Critical,
	"fatal":   Critical,
	"blocker": Critical,
	"致命":      Critical,
	"阻断":      Critical,
	"p1":      High,
	"major":   High,
	"urgent":  High,
	"p2":      Medium,
	"p3":      Low,
	"trivial": Low,
	"nit":     Low,
	"nits":    Low,
}

// Normalize 把任意写法归一到词表。
// 第二个返回值=false 表示原值越界（已归为 low，调用方应记录告警）。
//
// ⚠️ 三档宿主迁移警示（v0.8.0 四档化，oagentkit/textx#20）：v0.7 时 critical
// 会被折进 high（词表别名），switch/if 形态的消费点只列三档就够了；四档化后
// critical 保留原值——**只列 High/Medium/Low 分支的消费点会让 critical 静默
// 掉出所有分支**（既不进 high 计数、也不进 midLow 计数），最严重的意见落进
// 「其余 → 兜底」，而全仓测试若没有 NormalizeSeverity("critical")=="high" 这类
// 词表钉子（很常见），这个回归会无声上线（argus 消费侧实测踩坑）。三档宿主
// 升级后必须收编 critical 分支；暂不升四档刻度的宿主用 Normalize3 保持折叠
// 语义（显式选择，不靠别名静默兜底）。
func Normalize(s string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(s))
	if level, ok := aliases[v]; ok {
		return level, true
	}
	switch v {
	case "critical":
		return Critical, true
	case "high", "error", "严重", "高危":
		return High, true
	case "medium", "moderate", "warning", "warn", "中", "中等":
		return Medium, true
	case "low", "info", "minor", "hint", "低", "提示":
		return Low, true
	default:
		return Low, false
	}
}

// Normalize3 三档折叠语义的显式兼容钩子：与 Normalize 同一词表，但把
// critical 折进 high（v0.7 刻度），供暂不升四档刻度的宿主显式选择——
// 升级路径是收编 critical 分支后切回 Normalize，而不是让折叠语义继续
// 藏在别名表里。
func Normalize3(s string) (string, bool) {
	v, ok := Normalize(s)
	if v == Critical {
		return High, ok
	}
	return v, ok
}

// Rank 越大越严重（critical=4 / high=3 / medium=2 / low=1）。
func Rank(s string) int {
	switch s {
	case Critical:
		return 4
	case High:
		return 3
	case Medium:
		return 2
	default:
		return 1
	}
}

// Higher 取更严重的一方。
func Higher(a, b string) string {
	if Rank(a) >= Rank(b) {
		return a
	}
	return b
}

// Valid 是否属于钉死词表。
func Valid(s string) bool {
	return s == Critical || s == High || s == Medium || s == Low
}
