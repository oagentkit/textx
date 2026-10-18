# AGENTS.md — oagentkit/textx

> 文本与差异处理纯函数集，6 个包：`textutil` `diffx` `codeintel` `compact` `conversation` `reportutil`
> module：`github.com/oagentkit/textx` · org：[oagentkit = Open Agent Kit](https://github.com/oagentkit)
> 矩阵分层：**tier 0（基础层）**

## 本仓包含

```
textutil/  diffx/  codeintel/  compact/  conversation/  reportutil/
```

## 硬规则

### 分层与依赖

- 依赖严格自底向上，**禁止反向依赖**。本仓是 tier 0，任何矩阵内模块都不得被本仓依赖。
- 本仓只依赖 `github.com/oekit/pkg`（`stringx`/`encoding`/`mathx`）。**新增外部依赖前先确认
  `oekit/pkg` 里没有** —— 这是矩阵的既定分工。
- 新增依赖 = 抬地板，会拖慢所有下游，必须有明确理由。

### 错误处理

- 一律使用 `github.com/oekit/pkg/errors`：`errors.Wrap(err, "ctx", kv...)`（nil-safe）、
  `errors.Cause`；业务错误用 BizError。
- **类型同一性红线**：同一进程内只允许一套 errors 包。混用旧路径会把 BizError 的
  `As`/`Is` 判定打断。
- 所有 error 禁止 `_` 忽略。

### 日志

- **本仓不引入日志依赖。** 这是纯函数集，没有日志门面 —— 加日志等于给 tier 0 抬地板。
  需要观测输出时返回 error 或让调用方决定，不要在库里打日志。

### 并发安全

- 禁止裸 `go func`：用 `github.com/oekit/concurrency/async` 的 GoSafe 系列（panic 被 recover）。
- 共享状态用 mutex / atomic 保护；包级全局可变状态用 `atomic.Pointer[T]`。
- 本仓多数包应是**无状态纯函数**，新增可变全局前先自问是否该放到调用方。

### API 设计

- 函数选项模式命名 `XxxOption`（禁止裸 `Option`，多包 co-import 会冲突）。
- 导出符号 PascalCase + godoc 注释；包文档放 `doc.go`。
- 保持**零仓内依赖的纯函数形态**：能用返回值表达的错误不要用回调里的副作用。

### 版本与发版

- SemVer。module 路径 = `github.com/oagentkit/<name>`。
- **改 module 路径必须同步**：全部 import、go.mod、README 安装行。
- 0.x 阶段 minor 允许 breaking。

## 明确不属于本仓的

以下能力在矩阵别处，**不要在本仓重新实现**：

| 能力 | 归属 |
|---|---|
| 并发/流控/子进程/热插拔/钩子 | `github.com/oagentkit/runtime` |
| 熔断 | `github.com/oekit/concurrency/breaker`（`runtime/breaker` 已于 runtime v0.3.0 退役） |
| 通用 errors/stringx/encoding/mathx | `github.com/oekit/pkg` |
| 提示词组装/转向/文件状态 | `github.com/oagentkit/promptx` |
| 审计日志 | `github.com/oagentkit/guardx/audit` |
