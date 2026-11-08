# textx CHANGELOG

> 本文件由 `matrix-tools/scripts/gen_changelogs.py` 从 git 历史生成，按 tag 切段、取每段内的 conventional commit 渲染。

## [v0.10.2] — 2026-11-08

- 杂务(deps)：依赖地板追到最新——github.com/oekit/pkg v0.37.4

| 依赖 | 现地板 | 抬到 |
|---|---|---|
| `github.com/oekit/pkg` | `v0.37.3` | `v0.37.4` |
## 为什么
矩阵的**发版时刻地板时效**门禁（`verify_tag_floor --ref HEAD`）要求：各仓 HEAD
（= 下次发版会发出去的内容）的依赖地板，相对那一刻的依赖最新版本，不能已落后。
oekit 基座与矩阵模块都在高频发版 ⇒ 地板是**跑步机**，落后一格就会判红。
## 兼容性实测
本工具在提交前跑完 `go build` / `go vet` / `go test -race -count=1 ./...`（全绿才提交）
与 `go mod tidy -diff`（无漂移）⇒ 改动**只影响依赖版本**，零行为差异。
只改 `go.mod` / `go.sum` ⇒ 不需要 CHANGELOG 条目（版本段由下次发版的 `prewrite` 生成）。

- 杂务(deps)：依赖地板追平 oekit BOM v0.2.9——pkg v0.37.4 / logx v0.8.10 / concurrency v0.28.4

## [v0.10.1] — 2026-11-08

- 杂务(deps)：依赖地板追到最新——github.com/oekit/pkg v0.37.3

| 依赖 | 现地板 | 抬到 |
|---|---|---|
| `github.com/oekit/pkg` | `v0.37.1` | `v0.37.3` |
## 为什么
矩阵的**发版时刻地板时效**门禁（`verify_tag_floor --ref HEAD`）要求：各仓 HEAD
（= 下次发版会发出去的内容）的依赖地板，相对那一刻的依赖最新版本，不能已落后。
oekit 基座与矩阵模块都在高频发版 ⇒ 地板是**跑步机**，落后一格就会判红。
## 兼容性实测
本工具在提交前跑完 `go build` / `go vet` / `go test -race -count=1 ./...`（全绿才提交）
与 `go mod tidy -diff`（无漂移）⇒ 改动**只影响依赖版本**，零行为差异。
只改 `go.mod` / `go.sum` ⇒ 不需要 CHANGELOG 条目（版本段由下次发版的 `prewrite` 生成）。

- 杂务(build)：补统一 Makefile——本地 ci 门禁与共享 CI 同口径（agentkit#138）

Co-authored-by: Murphy-Yi <zhangyi@Murphy-YideMac-Mini.local>

- 杂务(deps)：依赖地板追到最新——github.com/oekit/pkg v0.37.1

| 依赖 | 现地板 | 抬到 |
|---|---|---|
| `github.com/oekit/pkg` | `v0.34.1` | `v0.37.1` |
## 为什么
矩阵的**发版时刻地板时效**门禁（`verify_tag_floor --ref HEAD`）要求：各仓 HEAD
（= 下次发版会发出去的内容）的依赖地板，相对那一刻的依赖最新版本，不能已落后。
oekit 基座与矩阵模块都在高频发版 ⇒ 地板是**跑步机**，落后一格就会判红。
## 兼容性实测
本工具在提交前跑完 `go build` / `go vet` / `go test -race -count=1 ./...`（全绿才提交）
与 `go mod tidy -diff`（无漂移）⇒ 改动**只影响依赖版本**，零行为差异。
只改 `go.mod` / `go.sum` ⇒ 不需要 CHANGELOG 条目（版本段由下次发版的 `prewrite` 生成）。
Co-authored-by: Murphy-Yi <zhangyi@Murphy-YideMac-Mini.local>


## [v0.10.0] — 2026-11-07

### 修复(compact)：microcompact 的节省量折算到 provider 口径——三个 token 数不再跨口径相减（#33）

## 根因
`maybeMicrocompact` 里 `before` 可能是 **provider 真实用量**（`usageTokens > 0`），
而 `saved` 来自 `applyClears`，是**本地估算**的增量。于是
`TokensAfter = before - saved` 是两种口径相减的杂交量。两个可判定的后果：
1. **三个数自相矛盾**：provider 说 5000、本地估出的节省 40 ⇒ 报「剩余 4960」，
   而真实剩余约 1000。`before - saved == after` 这个「自洽」是假的；
2. **门槛判错方向**（更坏）：`saved < MinTokenSavings` 用的也是本地口径。本地
   估算对中文系统性低估约 2 倍，真实能省几千 token 的清理在本地口径下只算几十，
   撞上 `DefaultMinTokenSavings(256)` 判 `below_savings` 不执行——**真 token 已
   超窗，轻量先手却在等一个永远不会到来的「够省」**。
附带项：`below_savings` 路径已填 `ClearedIDs` 却原样返回 `Messages`——报「清了
这些」而转写里找不到占位符。
## 修法
- `localBefore := EstimateTokens(msgs)` 始终留一份本地估算量做折算基准；
- 取到 provider 用量时，`saved = scaleToUsage(saved, localBefore, usageTokens)`
  ——按「本地节省 × provider 用量 / 本地总量」折算成同一口径，**先折算再比门槛、
  再相减**，三个数回到同一个量纲；
- 折算比的前提（被清条目与整段转写同 token 密度）写进 `scaleToUsage` 文档；
  取整向上（不得因取整把「省了一堆」折成 0）；`int64` 中间量防 32 位溢出；
- `ClearedIDs` 只在真正应用时填。
为什么不能反向折算：provider 只给了一个**整段**用量，没有「被清掉那几条各是多少」，
所以只能从本地增量往真实口径放大，不能反过来。
## 反向验证（两条，均实测红）
| 注入 | 报错 |
|---|---|
| 去掉 `saved = scaleToUsage(...)`（退回杂交口径） | `provider 口径下节省应按比例放大到门槛之上，实得 below_min_savings` |
| `below_savings` 路径仍填 `ClearedIDs` | `below_savings 不得报 ClearedIDs（转写原样返回却报「清了这些」）: [a]` |
## 用例（`compact/microcompact_oracle_test.go`）
- `TestMicrocompactUsageScalesSavingsIntoProviderOracle`：**成对**断言——同一份
  转写在本地口径下判 `below_savings`、在 provider 口径（×100）下必须
  `applied`，且 `saved` 恰为「本地节省 × 折算比」、`before - saved == after`。
  只断言有 usage 那条，一个把 saved 硬编码成大数的实现也能过；
- `TestMicrocompactNoUsageKeepsLocalOracle`：无 usage 时三个数全走本地口径且
  `saved == 转写前后估值差`（折算不得污染原有路径）。

### 修复(reportutil)：通道关联簇列表——共享签名但 eq 不同的等价项不再被拆成两簇各 1（#34）

* fix(diffx): `---` 头行也重置文件归属——截断块的 hunk 不再误归属给上一个文件（#31）
## 根因
包注释明确声明：
> `+++` / `---` / `diff --git` 行在任意位置都重置文件归属
实现只处理 `diff --git`（重置）与 `+++`（设归属），**从不看 `---`**。
于是截断/拼接形态（注释自称的兼容面）下，若某文件块只剩 `--- a/other.go`
头（`+++` 被截掉），其后的 hunk 会归属到**上一个文件**。消费方按行区间做
注入/定位就打错文件——而且是静默的：区间看起来完全合法，只是属于别的文件。
## 修法
头区状态机补 `--- ` 分支：命中即 `curFile = ""`。与声明对齐——hunk 归属
只认 `+++`，`---` 只负责把归属清空。`/dev/null` 侧不用特判（清空本来就是
它的效果）。
## 用例
新增 `diffx/hunk_minus_header_test.go`：
1. **`TestHunkLinesMinusHeaderResetsOwner`**——`a.go` 正常一块 + 只剩 `---`
   头的截断块 + hunk：断言 `a.go` 只有原始那一个区间、`other.go` 不入表、
   表里只有一项。
2. **`TestHunkLinesMinusHeaderNormalForms`**（成对对照）——完整块（`---` 清空
   → `+++` 设归属）、新增文件（`--- /dev/null`）、删除文件（`+++ /dev/null`
   不设归属）三种形态都不受新分支影响。
## 反向验证
去掉 `---` 分支：
```
--- FAIL: TestHunkLinesMinusHeaderResetsOwner
    hunk_minus_header_test.go:17: a.go 只该有原始那一个区间，
        实得 [[1 1] [10 11]]——截断块的 hunk 被误归属过来了
```
`[10 11]` 正是 `other.go` 那块的区间。
## 验证
`go test ./...` 全绿。
关联 oagentkit/textx#31
* fix(reportutil): 通道关联簇列表——共享签名但 eq 不同的等价项不再被拆成两簇各 1（#34）
## 根因
`Aggregate` 的 `byChannel` 是 `map[string]*Group[T]`——**每个通道只记首见簇**。
于是这个形态必然拆散：
- A、B、C 共享签名通道 `x`
- `eq(A,B)=false`、`eq(B,C)=true`
B 建了新簇，但它拿不到通道注册（`if _, seen := byChannel[ch]; !seen` 明确写着
「首见者占位，后到者不抢通道」）；C 走通道 `x` 撞上的是 A，`eq` 不过就自成
第三簇。结果 B/C 明明相互等价，却永远分居两簇，`Count` 各为 1。
聚簇的用途正是「多份采样相互复现 ⇒ Count≥2 升级权重」。这个形态下复现信号
被**静默拆散**：三份采样看起来像三个各出现一次的问题，前面反复出现的同一个
缺陷被降权。
## 修法
`byChannel` 改为 `map[string][]*Group[T]`——通道关联**簇列表**：
- 匹配轮：通道命中后，按注册顺序对该通道下的**每个**簇试 `eq`，命中即并入；
- 建簇轮：每个新簇都在自己的所有通道下登记（不再只让首见者占位）；
- 签名里重复出现的通道只登记一次（否则同一簇在同一通道下有多份，匹配轮白跑）。
文档同步：`eq` 的表述从「与首见者等价才并入」改成「与**某个**既有簇的代表
等价才并入，不做链式传递」——原句在修好之后已不成立，留着就是错的。
## 用例
新增 `reportutil/aggregate_channel_test.go`：
1. **`TestAggregateSharesChannelAcrossGroups`**（判别式）——`alpha`/`beta`/`beta`
   共享通道 `x`：断言 2 簇且 beta 的 `Count==2`。
2. **`TestAggregateStillJoinsFirstMatch`**（成对对照）——与首见簇等价时仍并入
   首见簇，「继续试其余簇」不得把明确等价的项推去另建新簇。
3. **`TestAggregateMultiChannelLookup`**——一个簇在 x/y 两个通道下都登记，
   后到者走任一路径都能找到它。
4. **`TestAggregateDuplicateChannelInSignature`**——签名里故意重复同一通道，
   结果不受影响（登记去重生效）。
## 反向验证
退回「每个通道只记首见簇」，复现的正是 issue 描述的三簇形态：
```
--- FAIL: TestAggregateSharesChannelAcrossGroups
    应为 2 簇（alpha 一通、beta 两通），实得 3 簇:
    [{...text:alpha... Count:1} {...text:beta... Count:1} {...text:beta... Count:1}]
```
两个 beta 各 `Count:1`。
## 验证
`go test ./...` 全绿。
关联 oagentkit/textx#34

### 修复(diffx)：`---` 头行也重置文件归属——截断块的 hunk 不再误归属给上一个文件（#31）

## 根因
包注释明确声明：
> `+++` / `---` / `diff --git` 行在任意位置都重置文件归属
实现只处理 `diff --git`（重置）与 `+++`（设归属），**从不看 `---`**。
于是截断/拼接形态（注释自称的兼容面）下，若某文件块只剩 `--- a/other.go`
头（`+++` 被截掉），其后的 hunk 会归属到**上一个文件**。消费方按行区间做
注入/定位就打错文件——而且是静默的：区间看起来完全合法，只是属于别的文件。
## 修法
头区状态机补 `--- ` 分支：命中即 `curFile = ""`。与声明对齐——hunk 归属
只认 `+++`，`---` 只负责把归属清空。`/dev/null` 侧不用特判（清空本来就是
它的效果）。
## 用例
新增 `diffx/hunk_minus_header_test.go`：
1. **`TestHunkLinesMinusHeaderResetsOwner`**——`a.go` 正常一块 + 只剩 `---`
   头的截断块 + hunk：断言 `a.go` 只有原始那一个区间、`other.go` 不入表、
   表里只有一项。
2. **`TestHunkLinesMinusHeaderNormalForms`**（成对对照）——完整块（`---` 清空
   → `+++` 设归属）、新增文件（`--- /dev/null`）、删除文件（`+++ /dev/null`
   不设归属）三种形态都不受新分支影响。
## 反向验证
去掉 `---` 分支：
```
--- FAIL: TestHunkLinesMinusHeaderResetsOwner
    hunk_minus_header_test.go:17: a.go 只该有原始那一个区间，
        实得 [[1 1] [10 11]]——截断块的 hunk 被误归属过来了
```
`[10 11]` 正是 `other.go` 那块的区间。
## 验证
`go test ./...` 全绿。
关联 oagentkit/textx#31

### 修复(diffx)：hunk 行数/起始行越界整块跳过——不再因整数下溢产出 end < start 的负区间（#32）

## 根因
`diffx.go` 的 hunk 解析把行数的解析错误丢掉了：
```go
cnt, _ = strconv.Atoi(m[2])
```
而 `hunkRe` 的 `(\d+)` 不限长度。`@@ -1,1 +1,99999999999999999999 @@` 时
`Atoi` 返回 `(MaxInt, ErrRange)`，错误被丢弃后 `start + cnt - 1` 整数下溢成
**end < start 的负区间**，下游按闭区间取行会取错或取空。
一段损坏（或恶意构造）的 diff 就能触发，且没有任何信号——这也是本仓
「所有 error 禁止 `_` 忽略」规则被违反的实际后果，不只是风格问题。
## 修法
两处：
1. **检查 `Atoi` 的 error**，越界即整块跳过（宁缺不错）。
2. **补加法溢出判据**：两个加数各自合法也可能在加法处溢出
   （`start = MaxInt` 解析成功、`cnt = 2` 也解析成功，但 `start+cnt-1` 越过 int）。
   判据写成减法 `cnt-1 > math.MaxInt-start`，避免判据自己溢出。
## 用例
新增 `diffx/hunk_overflow_test.go`：
1. **`TestHunkLinesOverflowSkipped`**——三条子用例：行数越界、起始行越界、
   加法溢出（两个加数各自合法）。断言 `m` 为空（整块跳过），并对任何残留区间
   断言 `end >= start` 且非负。
2. **`TestHunkLinesNormalRangesStillParsed`**（成对对照）——合法区间照常解析，
   且**边界但不溢出**的 `start=MaxInt, cnt=1 ⇒ [MaxInt, MaxInt]` 不得被误跳过
   （否则「一律跳过」也能让上一条绿）。
## 反向验证
退回旧实现（丢 error、无溢出判据），两条子用例红，其中一条复现的正是下溢：
```
--- FAIL: TestHunkLinesOverflowSkipped/行数越界
    hunk_overflow_test.go:48: 越界 hunk 应整块跳过，实得 map[a.go:[[1 9223372036854775807]]]
--- FAIL: TestHunkLinesOverflowSkipped/加法溢出（两个加数各自合法）
    hunk_overflow_test.go:40: a.go: 区间下溢 [9223372036854775807 -9223372036854775808]（end < start）
```
第二行就是 issue 描述的形态：`end` 变成了 `-9223372036854775808`。
## 验证
`go test ./...` 全绿。
关联 oagentkit/textx#32

### 修复(codeintel)：recvTypeName 识别泛型接收者——"Repo.Wrap" 不再退化为裸 "Wrap"（#36）

## 根因
`codeintel/callers.go` 的 `recvTypeName` 只认 `*ast.StarExpr` / `*ast.Ident` /
`*ast.SelectorExpr` 三支。泛型接收者在 AST 里不是这三者：单类型参数是
`*ast.IndexExpr`（`Repo[T]`），多类型参数是 `*ast.IndexListExpr`（`Pair[K, V]`），
落 default 返回 ""，于是 `funcDeclName` 退化成裸方法名。
`CallerRef.Caller` 的文档契约写的是「封闭函数（`Recv.Name` / `Name`）」
（callers.go:24）——接收者被静默丢弃，**两个不同泛型接收者的同名方法在调用方
归属上不可区分**。defs 侧不受影响（Recv 存的是源码文本切片），所以只有
callers 这一侧错，错得还不显眼。
## 修法
`recvTypeName` 补两个分支，都取 `.X` 递归——泛型实例化只影响类型实参，
不影响「方法属于哪个类型」这件事：
```go
case *ast.IndexExpr:
	return recvTypeName(t.X)
case *ast.IndexListExpr:
	return recvTypeName(t.X)
```
`.X` 递归而不是直接返回，是为了让 `*Repo[T]`（StarExpr→IndexExpr→Ident）、
`Repo[T]`（值接收者，IndexExpr 无 StarExpr 包裹）、`pkg.Repo[T]`
（IndexExpr→SelectorExpr）三种形态走同一条路径。
## 用例
新增 `codeintel/callers_generic_test.go`：
1. **`TestPreciseCallersGenericReceiver`**——判别场是「四个**同名** `Wrap`
   只有接收者形态不同，各自调用同一个 `Target`」：
   `*Repo[T]`（IndexExpr）、`*Pair[K, V]`（IndexListExpr）、`Val[T]`
   （值接收者）、`*Plain`（非泛型对照）。断言四种 Caller
   `Repo.Wrap` / `Pair.Wrap` / `Val.Wrap` / `Plain.Wrap` 各一条，
   且**不出现裸 `Wrap`**——裸名正是「接收者被丢弃」的具体形态。
2. **`TestPreciseCallersGenericReceiverCallee`**——泛型方法体内的方法调用
   照常解析：`Caller="Repo.Wrap"` 且 `Callee="r.Save"` 同时对。补接收者归属
   不该让 Callee 一起退化（两者走的是不同的分支）。
## 反向验证
退回修复（删掉两个 case）后，用例逐条复现 issue 里的症状：
```
缺 Caller="Repo.Wrap"（实得 map[Plain.Wrap:1 Wrap:3]）
缺 Caller="Pair.Wrap"（实得 map[Plain.Wrap:1 Wrap:3]）
缺 Caller="Val.Wrap"（实得 map[Plain.Wrap:1 Wrap:3]）
出现裸 Caller="Wrap" 3 次——接收者类型名被丢弃（实得 map[Plain.Wrap:1 Wrap:3]）
Caller 去重后 2 种，应为 4 种
Caller="Wrap"，应为 "Repo.Wrap"          ← Callee 用例
```
`Wrap:3` 就是 issue 复现里那条 `Caller="Wrap"`：三个泛型接收者塌成同一个
裸名，而**非泛型对照 `Plain.Wrap` 一直是对的**——所以缺陷看起来像
「泛型不参与归属」，实际是 `recvTypeName` 的穷举漏了两支。
## 影响面
`argus` 的 `impactview/symbols.go`（:79-101）是同款复制实现，有一模一样的缺陷，
随上游修复同步更新。
关联 oagentkit/textx#36

### 修复(diffx)：headerPath 先 unquote 再剥侧前缀，真实 git quoted 形态不再残留 b/（#35）

根因：git 的 quote_two 语义把**侧前缀与路径整体**加引号（`+++ "b/a\"b.go"`），
而 headerPath 是「先 TrimPrefix(side) 再查引号」。首字符是 `"` ⇒ 剥前缀永不命中，
unquote 后 `b/` 就跟着留在结果里。
不是边角情况：git 出厂 core.quotepath=true，**任何非 ASCII 文件名**都输出成
`+++ "b/\346\226\207\344\273\266.go"`。残留 b/ ⇒ 下游按真实仓库路径查 map 必落空
⇒ 该文件的 diff **静默丢失**（argus 的 fillFromRawDiff 实测是这个形态，
PrepareIncremental 还会因此永远判「未审过」、增量去重整体失效）。
修法：两种引号形态各判一次，而不是调换顺序——
- 引号**整体包住前缀+路径**（真实 git 形态）⇒ 先 unquote 再剥前缀；
- 引号只包路径本体（`+++ b/"a b.go"`，v0.12.0 注释假设的形态）⇒ 先剥前缀再 unquote。
  真实 git 不产生后者，但既有行为有用例覆盖，不能因为调换顺序把它弄坏。
抽 isQuoted 统一两处的判定形态，顺带消掉重复的 HasPrefix/HasSuffix 条件。
反向验证：把实现逐字退回 v0.12.0（先剥前缀→再查引号）后，
`+++ "b/\346\226\207\344\273\266.go"` 得到 `b/文件.go`（与 issue 描述逐字一致），
9 条子用例里 5 条红；恢复后全绿。
用例覆盖：引号内含侧前缀 / 非 ASCII 八进制转义 / --- 侧 a/ / 引号+行尾附加信息 /
引号+CRLF / 前缀在引号外（既有形态不回退）/ 裸路径 / --no-prefix / quoted 无前缀，
外加 DiffPath 端到端两条（含删除文件回退 --- 侧）。

- 杂务(deps)：依赖地板追到最新——github.com/oekit/pkg v0.35.2

- 文档(textutil)：两套 token 口径写明适用场景与约 2 倍量纲差，并用例钉住倍率（#30）

## 根因（口径断裂的库侧）
`EstTokens`（纯 rune，rune/3）与 `EstTokensCJK`（汉字 2 单位，units/3+1）在
**同一段中文文本上差约 2 倍**，而两者都在公开面上、都没有写明适用场景——
`EstTokens` 的文档只有「已知 rune 数时的 token 估算（向上取整）」一句。
后果是消费方各挑一个：`llm` 的预算记账用 CJK 口径，`promptx` 的段计量用纯 rune
口径（后者已由配套 PR 切正），同一条链上「组装时报告 800、发出去预算说 1600」，
预算闸门与成本看板一起失真。
这不是精度问题——**是量纲问题**：两种口径的分母不同，混用时比值就是错的，
而且估错不崩，只会静默挤掉内容或提前拒绝。
## 修法（只动文档与用例，不改行为）
- `EstTokens` / `EstTokensOf` 文档补「纯 rune 口径，对中文系统性低估约 2 倍」
  + **该用哪个**的对照表（ASCII / 含中文 / 需与 llm 对齐 / 只有 rune 计数）；
- `EstTokensCJK` 文档补「跨模块口径的唯一推荐值」与「短文本略偏大是刻意的取舍」；
- 行为一行未改——`EstTokens` 仍是 rune/3，`EstTokensCJK` 仍是 units/3+1，
  既有调用点零影响。
## 为什么要有用例
文档里写「约 2 倍」是散文，散文会漂：有人改了除数或 `+1`，注释照旧写着 2 倍。
既然跨模块预算记账靠的就是这个倍率关系，就把它钉成断言。
- `TestEstTokensOracleRatioOnPureCJK`：300/600/1500 个汉字上 CJK 口径 / 纯 rune
  口径的比值必须落在 1.8~2.4。区间而非等号（两函数各有取整与 +1，小样本有抖动），
  但「接近 2」与「接近 1」是口径分水岭，判得开；
- `TestEstTokensOracleAgreeOnASCII`：成对对照——纯 ASCII 上两口径只差取整与 +1，
  所以中文场景的 2 倍差**不能**用「反正估不准」糊过去；
- `TestEstTokensCJKRunesMatchesCJK`：`EstTokensCJKRunes` 与字符版同口径（它存在的
  理由是免物化字符串，不是另一套算法）。
## 反向验证（实测红）
把 `EstTokensCJK` 的 `if runeUnitsCJK(r) == 2 { units += 2 }` 改成 `units++`：
```
300 个汉字：纯 rune 口径 100、CJK 口径 101，比值 1.01 —— 文档声明约 2 倍；
比值落到 1 附近说明 CJK 口径被改回了纯 rune
```


## [v0.9.0] — 2026-11-06

- 文档(reportutil)：Normalize 补三档宿主迁移警示 + 新增 Normalize3 兼容钩子

v0.8.0 四档化（#20）语义正确，但 v0.7 三档宿主升级时有一处静默迁移坑
（argus 消费侧实测）：switch/if 形态的消费点只列 High/Medium/Low 分支时，
critical（v0.7 折进 high、v0.8 保留原值）会静默掉出所有分支——最严重的
意见落进「其余 → 兜底」，没有词表钉子测试的宿主抓不到这个回归。
- Normalize doc 补 ⚠️ 迁移警示（坑的形态、为什么测试抓不到、两条出路）
- 新增 Normalize3 显式兼容钩子：critical→high 折叠语义供暂不升四档刻度
  的宿主显式选择（升级路径：收编 critical 分支后切回 Normalize）
- TestNormalize3FoldsCritical：折叠正确性 + 越界 bool 语义 + 三档封闭
关联 oagentkit/textx#27
Co-authored-by: Murphy-Yi <zhangyi@Murphy-YideMac-Mini.local>


## [v0.8.0] — 2026-11-05

### 新增(reportutil)：severity 四档化——critical 独立顶格档（#20）

此前 critical/fatal/blocker 全部折进 high、Rank 刻度只有 3/2/1，四档宿主
（审批闸按档位裁决）被迫在消费侧保留 critical 特判残根，忘了特判就与
high 静默同档、顶格语义丢失。
- 新增 Critical 常量；Normalize/aliases/Rank/Higher/Valid 同步四档
- Rank 刻度 4/3/2/1：high/medium/low 相对序不变，仅顶格档上移
- P0-P3 别名按 4 档刻度取自然映射（P0→critical/P1→high/P2→medium/P3→low）
  ——P0 与 P1 之间不再空一档
- fatal/blocker/致命/阻断 归 critical（顶格语义：crash/release blocker）
- 新增 TestNormalizeOutputAlwaysInVocab：Normalize 输出恒属钉死词表，
  下游 switch/查表不需要第四分支特判
Co-authored-by: Murphy-Yi <zhangyi@Murphy-YideMac-Mini.local>

- 杂务(deps)：依赖地板追到最新——github.com/oekit/pkg v0.34.1

| 依赖 | 现地板 | 抬到 |
|---|---|---|
| `github.com/oekit/pkg` | `v0.34.0` | `v0.34.1` |
## 为什么
矩阵的**发版时刻地板时效**门禁（`verify_tag_floor --ref HEAD`）要求：各仓 HEAD
（= 下次发版会发出去的内容）的依赖地板，相对那一刻的依赖最新版本，不能已落后。
oekit 基座与矩阵模块都在高频发版 ⇒ 地板是**跑步机**，落后一格就会判红。
## 兼容性实测
本工具在提交前跑完 `go build` / `go vet` / `go test -race -count=1 ./...`（全绿才提交）
与 `go mod tidy -diff`（无漂移）⇒ 改动**只影响依赖版本**，零行为差异。
只改 `go.mod` / `go.sum` ⇒ 不需要 CHANGELOG 条目（版本段由下次发版的 `prewrite` 生成）。
Co-authored-by: Murphy-Yi <zhangyi@Murphy-YideMac-Mini.local>

- 杂务(deps)：依赖地板追到最新——github.com/oekit/pkg v0.34.0

oekit/pkg v0.34.1 含 StripCodeFence 语言标记修复（pkg#180）——本仓
textutil.StripCodeFence 委托上游，抬地板即收编（oagentkit/textx#21）。
Co-authored-by: Murphy-Yi <zhangyi@Murphy-YideMac-Mini.local>


- 格式(reportutil)：gofmt 对齐 severity 注释缩进

Co-authored-by: Murphy-Yi <zhangyi@Murphy-YideMac-Mini.local>


## [v0.7.2] — 2026-11-04

### 重构(textutil)：围栏剥离委托 oekit pkg/jsonrepair.StripCodeFence——围栏语义单源上游（本包此前为逐字副本，多仓并存各自漂移）；deps 地板 pkg v0.31.2

- 9649258

- 杂务(deps)：依赖地板追到最新——github.com/oekit/pkg v0.31.0

oekit 基座发了新版本（pkg v0.31.0 / concurrency v0.23.0 / logx v0.8.0）⇒ 本仓的
依赖地板落后 ⇒ 抬到最新，让**下一次发版**不再带出落后地板（打 tag 时刻的地板
必须是当时的最新——`verify_tag_floor` / `release.py doctor` 都在查这条）。
兼容性实测：本仓只用这些基座包的少量 API ⇒ go build / go vet / go test -race
全绿，**零行为差异**；go 指令未被顶起（本仓的 go 地板不动）。


## [v0.7.1] — 2026-10-26

- 杂务(deps)：依赖地板追平上游本轮新版本——oekit/pkg v0.28.0

v0.7.0 打 tag 时本仓 go.mod 钉的是 oekit/pkg v0.20.0，而那时上游最新已是 v0.28.0
⇒ 该 tag 的「发版时刻地板时效」不达标（matrix-tools 的 Release floor 门禁判红）。
tag 不可重打 ⇒ 只能靠后续版本补齐。本版只抬基座地板，不改任何行为：
- github.com/oekit/pkg v0.20.0 → v0.28.0


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
