package textutil

import (
	"github.com/oekit/pkg/jsonrepair"
)

// StripCodeFence 剥离 markdown 代码围栏（全仓**代码围栏**语义单一事实源：
// jsonrepair 宽容解析与快路径切片共用）。取**第一个** ``` 围栏块；无栅栏原样
// TrimSpace 返回。多围栏块输出（模型补多段代码）语义为「取第一块」——与宽容
// 解析链一致。注意与 fence 包的「数据区围栏」（fence.Data，注入卫生面）是两个
// 正交概念：本函数处理 markdown ``` 栅栏，后者圈定不可信数据边界。
//
// 实现委托 oekit/pkg/jsonrepair.StripCodeFence：围栏语义的**单源在上游**
// （oekit/pkg#74 已把该原语导出到 pkg/jsonrepair），本包此前是逐字相同的副本
// ——三份副本（agentkit / agentkit-bisect / textx）会各自漂移。委托后口径要改
// 只改上游一处。
func StripCodeFence(s string) string {
	return jsonrepair.StripCodeFence(s)
}
