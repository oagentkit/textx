# textx

文本与差异处理纯函数集 —— oagentkit 模块矩阵 **tier 0**（基础层，零仓内依赖）。

```
go get github.com/oagentkit/textx
```

| 包 | 职责 |
|---|---|
| `textutil` | 按行边界把文本切成增量片：多行凑片 ≤target(rune) 且不跨行，单行超 target 整行成片（硬上限 hard）⇒ 每片 ≤ max(target, hard)；流式分片场景的底座 |
| `diffx` | unified diff 解析纯函数 |
| `codeintel` | 轻量代码符号索引 |
| `compact` | 会话压缩策略面：auto-compact 判定（阈值/熔断/token 双轨）+ 压缩/恢复 |
| `conversation` | 多轮会话历史原语：滚动窗口切分、历史渲染 |
| `reportutil` | 评审/评测报告后处理：severity 归并、sampling、stats、`Fingerprint`/`NormalizeComment` 精确指纹 |
| `budget` | 上下文窗口输入侧预算：分段计量 + `MaxInput` 窗口钳位 + 压实计划骨架（三重空转守卫；摘要生成与原文落盘留宿主接口）。自 goagent-kernel 平移（归属评测 2026-11-26：对话域上下文工程，判据主语不是编排） |
| `spill` | 输出预算溢出存档：提示契约 / 分页取回（**rune 无损**，页边界不切坏多字节字符）/ 按龄清理 |
| `offload` | 内联大对象外置：sha 去重 + **两阶段**（首次标记、后续轮才替换占位）+ fail-soft；`ExtForMimeType` 显式 override 表防 `.jfif` 静默改键 |

## 定位

**tier 0，依赖方向的最底层。** 矩阵内没有任何模块可以反向依赖本仓，
本仓也不依赖任何矩阵内模块 —— 只依赖 `github.com/oekit/pkg`。

## 与 oekit 的关系

本矩阵（oagentkit，Agent/LLM 编排工具链）**依赖** oekit（Open Easy Kit，Go 微服务工具链）：

- `github.com/oekit/pkg` —— `stringx` / `encoding` / `mathx`

oekit 是另一套独立矩阵，不是本仓的子品牌。

## 开发

```bash
go test ./... -race
go vet ./...
gofmt -l .
```

规范见 [AGENTS.md](AGENTS.md)。

## License

MIT
