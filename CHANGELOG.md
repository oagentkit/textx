# textx CHANGELOG

> 本文件由 `matrix-tools/scripts/gen_changelogs.py` 从 git 历史生成，按 tag 切段、取每段内的 conventional commit 渲染。

## [v0.7.0] — 2026-10-26

### 新增(codeintel)：PreciseCallers 调用位精化——token 候选 + AST 封闭函数/限定名

- 与 GoCallers 同候选源（calls 表，定义行剔除），AST 二次核实只减不增：
  排除非调用位、补 Caller（Recv.Name/Name，包级初始化记 "(package init)"）
  与 Callee（pkg.F / T.M / F）
- 零磁盘 IO：候选文件源码建索引期已驻留（fileEntry.src），内存重解析；
  查询侧并发安全（不写 Index 字段）
- 边界如实声明：无类型解析——同名异接收者方法全命中、接口分派不可见，
  非 SSA 精确调用图
- 对照测试钉死与 GoCallers 的同源契约（候选集一致，精化只减不增）

## [v0.6.2] — 2026-10-21

- 杂务(deps)：依赖地板追平上游本轮新版本——oekit/pkg v0.20.0

## 下游要做什么
**`go mod tidy`**。地板抬高对下游是隐式 breaking：不跑就报
`updates to go.mod needed`，且报错不提真正原因（**不是本仓的 API 变了**）。
本仓**自身代码零改动**——只看本仓的 diff 看不出这是一次地板抬高。
## 抬了哪几块、为什么无行为差异
**`github.com/oekit/pkg` v0.18.0 → v0.20.0**
上游 diff 只动了 5 个包：
- `errors`：**纯增量**——新增 `Errorf`（格式化 + `%w` 包装，nil-safe）。
  `New`/`As`/`Is`/`Unwrap`/`Wrap` 是同型透传，一字未动。
- `httpjson`：**退避溢出修复**——旧写法 `base << (attempt-1)` 在 attempt 较大时
  int64 溢出成负数，负数绕过 `MaxDelay` 封顶，`time.After(负数)` 立刻返回
  ⇒ **该轮退避彻底消失**。改为乘法 + 饱和封顶。正常 attempt 数（1..20）下
  结果**逐字相同**，差别只在溢出的病态路径。
- `urlguard`：**纯增量**——新增 `WithExactHosts`，未设置时行为与既有完全一致。
- `stringx`：**仅注释**（补 `TruncEllipsis` 的 n 语义说明）。
- `pathx`：新增包（`SanitizeRel` 等）。
**本仓实际 import 的 `oekit/pkg` 子包：encoding mathx stringx **
与上游改动相交的：**stringx**
## 门禁
```
go build / vet / test -race   ✅
gofmt -l .                    ✅ 空
```

- 杂务(deps)：oekit 依赖地板抬到 v0.18.0

oekit 姊妹矩阵的依赖地板对齐到最新：
- github.com/oekit/pkg        v0.17.0 → v0.18.0
- github.com/oekit/concurrency v0.17.1 → v0.18.0
- github.com/oekit/logx       已是最新 v0.7.1，不动
oekit v0.18.0 的两个发布都是**纯增量**，无破坏性变更：
- pkg：jsonrepair 补抽数组原语 ExtractArray；
- concurrency：新增 `DoValue[T]`（带返回值的重试），`Do` 改为委托给它——
  退避序列、抖动、MaxDelay 封顶、DelayHint 优先、Retryable 门控、ctx 取消
  **只有一份实现**（此前这类调用方只能自己重写一遍退避，且常丢掉 DelayHint 语义）。
依据 oagentkit/matrix-tools#44 的还债流程：依赖地板对齐到发版时刻的最新。
本仓本轮不发 tag（不在发版集内），地板在 main 上就位，下次发版自然带上。


- 文档(AGENTS)：在「版本与发版」小节内补「发版顺序」指针

textx 是 13 仓里唯一已有「### 版本与发版」小节的仓，本次**合并进那一节**
（新增 #### 发版顺序 子小节），不并排出第二个同级小节。
13 个业务仓的 AGENTS.md 此前对发版顺序零覆盖（只有 textx 有个
「### 版本与发版」标题，但里面只讲 SemVer 与 module 路径），
发版顺序只写在 org 规则仓的 .github/CONTRIBUTING.md 里，单仓读者看不到。
本次只加**指针 + 本仓相关提醒**，runbook 本体保持**单一副本**
（CONTRIBUTING §发版），避免 13 份副本各自漂移。
覆盖四条：
- CHANGELOG.md 由 matrix-tools/scripts/gen_changelogs.py 从 git 历史生成，不要手改
- tag 必须落在 CHANGELOG 提交之后（下游 go get 拿到的是 tag 快照）
- 警告：无参重生成会吞掉待发版段，--next-version 是唯一能产出待发版段的开关
- 本仓相关提醒（见正文）

- 文档(changelog)：按新生成器规则重生成 CHANGELOG.md

生成器修复（oagentkit/matrix-tools#43）落地：
① `[round-N] ` 前缀被剥离 → conventional commit 解析不再失效，
   **commit body 保留**（此前整段被丢弃，发布说明只剩标题）；
② 非 conventional 提交也保留 body；
③ `--check` 识别「待发版段」（预写了但还没打 tag 的那段）。
本 PR 恢复的是**本就在 commit body 里**的正文（迁移步骤、代码块、
判据锚点）——此前被 `[round-N] ` 前缀导致的解析失败丢弃了。
**没有任何发布说明内容被删除**：删除的行都能在新增行里找到对应
（规范化掉 `(#N)` 与 `[round-N] ` 前缀后逐行比对）。
关联：oagentkit/matrix-tools#43、oagentkit/matrix-tools#40


## [v0.6.1] — 2026-10-20

### 修复(go)：tier0 基础包地板降到 go 1.24.0，CI go-version 同步到 1.24

本仓是 tier0 基础包（零重量级第三方地板，可被零依赖核心安全依赖），
但 go 指令声明的是 1.26.0——地板被无谓抬高，把 Go 1.24 的用户挡在门外。
## 依据
**依赖链全是 1.24 的包**：本仓唯一依赖 <base>（go 1.24.0），
依赖链上没有任何要求更高版本的包。
**代码本身兼容 1.24**：用真实 go1.24.0 工具链（`GOTOOLCHAIN=go1.24.0`）
build + test 全绿——不是推断，是实跑。
**门槛是 go 指令，不是 toolchain**（编译器实证）：
- `go 1.28.0` → 被拒：`go.mod requires go >= 1.28.0 (running go 1.27.1; GOTOOLCHAIN=local)`
- `go 1.24.0` + `toolchain go1.28.0` → 通过
## toolchain 行的取舍：删掉
toolchain **不构成门槛**（上面已实证），但 `toolchain go1.27.2` 会让 `go` 命令
**自动下载** 1.27.2，与「最低支持版本是 1.24」的精神相悖——用户以为自己用
1.24 在编译，工具链却被悄悄换成了 1.27.2。故删除该行。
这也与基座 org 的做法一致：oekit 的 pkg / concurrency / core / logx / metrics /
kitx 六个基础仓一律是 `go 1.24.0` + **无 toolchain 行**。
## 改动
- `go.mod`：`go 1.26.0` → `go 1.24.0`；删 `toolchain go1.27.2`
- `.github/workflows/ci.yml`：`go-version: '1.26'` → `'1.24'`
CI 的 go-version 与 go.mod 的 go 指令**成对**——这是跨层通用纪律，不因本次
降级而放松。CI 用高版本构建、go.mod 承诺低版本时，CI 全绿但用户按最低版本
编译失败，门禁绿了而承诺没兑现。
## 未受影响
- `require` 段（依赖地板钉）**未动**，`go mod tidy -diff` 零差异、go.sum 未变
- 内部 require 版本不变 ⇒ 不影响任何下游的 MVS
- 对外产物（versions.json / llms.txt）不记录 go 版本，无需刷新
## 验证
- `GOTOOLCHAIN=go1.24.0 go build ./...` → exit 0
- `GOTOOLCHAIN=go1.24.0 go test ./...` → 全绿
- `go mod tidy -diff` → 零差异
门禁见 oagentkit/matrix-tools PR：`scripts/verify_go_floor.py`
（tier0 必须 1.24.x + CI go-version 与 go.mod 成对，判据独立不复用生成器）。

### 修复(agents)：审计日志归属改为 obs/audit（guardx/audit 已迁出）

AGENTS.md 的「能力 → 包」映射表把审计日志指向 github.com/oagentkit/guardx/audit，
但该包已于 guardx v0.7.0 迁出（现为 github.com/oagentkit/obs/audit），不存在。
AGENTS.md 是 AI Agent 最常读的指引文件，照着 import 会 module not found。
本仓代码并未 import guardx/audit（go.mod 无 guardx 依赖），是纯文档错误归属。
按同表既有用法补注迁出说明（原 runtime/breaker 行同款写法）。
发现途径：oagentkit/.github 的 Matrix docs drift 门禁
(scan_stale_refs.py) —— 它同时抓出了这条真 bug 与三处误报，
误报部分已在 oagentkit/matrix-tools#(下一 PR) 修掉。

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

- 杂务(deps)：依赖地板对齐远端最新（pkg v0.15.1）

matrix-tools 的 verify_dep_floors.py 首次全矩阵扫描查出：43 条第一方依赖里
28 条落后，横跨 11 个仓；本仓是其中之一。
所升版本与本仓原钉版本相比源码零改动，逐版实测确认：
- textx v0.3.0→v0.4.0 / obs v0.4.1→v0.5.0 / runtime v0.5.0→v0.6.0 /
  mcpkit v0.5.1→v0.6.0  全树逐字节相同（diff -rq 退出码 0）
- oekit/pkg v0.14.0→v0.15.1  仅新增 urlguard 包（纯标准库，pkg 的 go.mod
  连 require 块都没有，零新增三方依赖）
- oekit/concurrency v0.17.0→v0.17.1  仅三个文档文件
- oekit/logx v0.7.0→v0.7.1  仅 CI 配置与文档
无 API 变更、无兼容风险。
本地门禁：go mod tidy -diff 无输出 / go build / go vet / gofmt / go test -race 全通过。


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

### 修复(agents)：熔断归属改为上游，runtime/breaker 已退役

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

- 杂务(deps)：oekit 依赖升级

pkg v0.13.0→v0.14.0
姊妹矩阵 oekit 的依赖地板此前被漏掉（上一轮只扫了 oagentkit 内部依赖），本次一并对齐到最新。
升级前已确认 oekit/concurrency v0.13.1..v0.16.0 的 8 个提交全是 feat/docs/chore，
无 breaking/remove 标记。其中 #28 订正了 `WithCooldown(0)` 的语义——
即 llms.txt 里警告过的那个陷阱，升上去正好拿到修正后的行为。
go.sum 的 h1 取自 sumdb，仅就地替换目标 mod 的两行、其余行不动。

- 杂务(deps)：oekit 三模块升至最新（pkg v0.13.0 / concurrency v0.12.0 / logx v0.7.0）

拿到上游新增能力：
- concurrency #22 RetryableWithHeaders：403 头感知判定（次级限流 / 主限流耗尽）
- concurrency #23 FanInLimit：受限并发扇出
- concurrency #24 WithTokenClock、#21 Lockout（keyed lockout/penalty）
- logx #12 TraceHandler：给裸 slog.Handler 注入 trace_id
破坏性变更核对（均无影响）：
- logx v0.7.0 ParseLogLevel 签名改为 (Level, error) —— 我们未使用该函数
- concurrency v0.9.0 的 Slot.Panicked 废弃 —— rag 已迁到 Slot.Err + errors.As
门禁：go build / go vet / go test -race 全绿。


- CI：接入 org 共享门禁（规则本体在 oagentkit/.github）
