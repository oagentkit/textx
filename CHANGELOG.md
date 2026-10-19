# textx CHANGELOG

> 本文件由 `matrix-tools/scripts/gen_changelogs.py` 从 git 历史生成，按 tag 切段、取每段内的 conventional commit 渲染。

## [v0.6.0] — 2026-10-19

### 新增(textutil)：token 折算系数可配 + 检索用词元切分原语

修 #3 与 #4。
一、EstTokensCJK 的折算系数可配（#3）
- CJKRatio{CJKCharsPerToken, ASCIICharsPerToken} + DefaultCJKRatio()（CJK 2 / ASCII 4）
- EstTokensWithRatio(s, r) / EstTokensWithRounding(s, r, mode)
- 取整方向**参数化**：RoundingSegment（逐段，默认）/ RoundingTotal（总量）
  两者在混合文本上差系统性（"中a中a中a中a"：8 vs 3）
- 非法系数回退默认，不 panic
- EstTokensCJK 签名与行为**保持不变**（老调用点不受影响）
为什么值得单独做：EstTokensCJK 是「单位数/3+1」写死折算，折出来 ≈ CJK 1.5 字/token、
ASCII 3 字符/token，与 octop 口径（CJK 2 字/token、ASCII 4 字符/token）差约 1.7 倍。
预算类场景估错不崩，但会**静默挤掉**别的内容（1500 token 预算能塞的内容差近四成）。
二、Tokens：CJK 逐字 + 拉丁按词切（#4）
- Tokens(s) []string：汉字/假名/谚文逐字成词元，拉丁数字按空白与标点切词
- TokensJoined(s)：索引侧（入库前先切成空格分隔的词元串）
- FTS5Query(s)：查询侧，每个词元加引号 → FTS5 的 AND/OR/NOT/NEAR 降回字面量，
  任意用户输入都不构成语法错误、也不会被当成 NOT 条件
- 语义边界写进 godoc：词的合集、不是连续子串；不支持短语检索；
  与 BigramSet 并存不互相替代（阈值是按词元口径标定的，换二元组口径会翻转判定）
为什么需要：FTS5 的 unicode61 把连续汉字当**一个词元**，「磁盘」与「磁盘长期偏小」
永不相等 → 0 命中，且**静默**返回空集（用户看到的是「库里没有这条」）。

## [v0.5.0] — 2026-10-19

- 杂务(deps)：依赖地板对齐远端最新（pkg v0.15.1） (#13)

## [v0.4.0] — 2026-10-18

- 与 v0.3.0 相比无代码差异（版本号/打包变更）

## [v0.3.0] — 2026-10-18

### 新增：textx 仓建立——文本/差异/会话/报告纯函数集（6 包）

从 agentkit 单仓切出 tier0 基础层模块，module 路径 github.com/oagentkit/textx。
收编 6 个包：
- textutil     按行边界增量切片，凑片不跨行
- diffx        unified diff 解析纯函数
- codeintel    轻量代码符号索引
- compact      会话压缩策略面（阈值/熔断/token 三轨判定）
- conversation 多轮会话历史原语（滚动窗口切分/历史渲染）
- reportutil   评审报告后处理（severity/sampling/stats/Fingerprint）
外部依赖仅 github.com/oekit/pkg（stringx/encoding/mathx），无矩阵内依赖。
门禁：go build / go vet / go test -race 6 包全绿，gofmt 干净。

### 修复(agents)：熔断归属改为上游，runtime/breaker 已退役 (#1)

`AGENTS.md` 的「明确不属于本仓」表仍把熔断归给
`github.com/oagentkit/runtime/breaker`，标注「待退役到 oekit/concurrency/breaker」。
但 `runtime` 仓已于 v0.3.0 **移除**该包 —— 不是「待退役」，是已经退役了。
`AGENTS.md` 是 AI Agent 读得最多的文件。Agent 照着这行去 import 会拿到
module not found，而表格里其它行都合法，整张表看不出异常。
改为指向真实存在的上游包，并标明退役版本：
    | 熔断 | `github.com/oekit/concurrency/breaker`（`runtime/breaker` 已于 runtime v0.3.0 退役） |
配套门禁：matrix-tools 的 `scan_stale_refs.py` 会扫出这类「看起来合法但
包已不存在」的引用，已接入 org CI。

### 修复：补回 go.sum（搬移工作区时遗漏，导致 CI 门禁 missing go.sum entry）

- 9d4fa38

- 杂务(deps)：oekit 依赖升级 (#2)
- 杂务(deps)：oekit 三模块升至最新（pkg v0.13.0 / concurrency v0.12.0 / logx v0.7.0）

- CI：接入 org 共享门禁（规则本体在 oagentkit/.github）
