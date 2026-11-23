// langcap.go —— 语言×能力矩阵单一事实源（oagentkit/textx#107）。
//
// 「哪个能力覆盖哪些语言（哪些扩展名、什么精度档）」的程序面唯一出处：
// 词法扫描三面（图层 / 函数级 diff / 语义分块）的参与扩展名集由本文件声明，
// 消费方（下游宿主/工具）直接引用；工具描述口径经 LangsOf 派生，README
// 语言矩阵与本表同源。
//
// 族（Family）与能力面（*Exts）分层声明：能力面可由族并集派生的走派生
// 保持真单源；语义分块面是独立历史面（.pyw/.mjs/.cjs 等不参与），刻意
// 不并集——同一语言在不同能力的参与面不同，收敛的是「声明处」，不是覆盖
// 面本身。语言族内部的解析器（正则 / go/parser / go/types）留在各消费方，
// 本文件只管覆盖面与口径。

package codeintel

import "strings"

// 语言族扩展名（词法面的最小声明单元）。
var (
	GoExts    = []string{".go"}
	ShellExts = []string{".sh", ".bash"}
	PyExts    = []string{".py", ".pyw"}
	JSTSExts  = []string{".js", ".mjs", ".cjs", ".jsx", ".ts", ".tsx"}
	RustExts  = []string{".rs"}
	JVMExts   = []string{".java", ".kt"}
	CExts     = []string{".c", ".h", ".cpp", ".hpp", ".cc", ".cxx"}
	SQLExts   = []string{".sql"}
)

// 词法扫描能力面（能力×语言矩阵的程序面）。
var (
	// ScanExts 词法图层扫描参与面：shell source/fn 调用、py·js·ts import、
	// rust mod/use、jvm import、c-family include、SQL 表血缘；Go 走
	// go.mod import 闭包与 AST（不在此列）。
	ScanExts = concatExts(ShellExts, SQLExts, PyExts, JSTSExts, RustExts, JVMExts, CExts)

	// FuncDiffExts 词法函数级 diff 参与面：shell fn / py def / rust fn /
	// js·ts 函数与类方法。Go 走 go/parser AST 版（funcdiff.go），不在此列。
	FuncDiffExts = concatExts(ShellExts, PyExts, RustExts, JSTSExts)

	// ChunkExts 语义分块索引参与面：Go=声明级 chunk+调用边；其余=文件级
	// 摘要 chunk（代码为主 + 仓库文档——.md 常载规范与架构语义）。独立
	// 历史面：.pyw/.mjs/.cjs/.bash/.cxx 等不在集内，扩面前先评估分块预算。
	ChunkExts = []string{
		".go", ".md", ".js", ".ts", ".jsx", ".tsx",
		".py", ".java", ".rs", ".c", ".cc", ".cpp",
		".h", ".hpp", ".rb", ".php", ".cs", ".kt",
		".swift", ".scala", ".sh", ".sql", ".yaml",
		".yml", ".toml", ".proto", ".vue",
	}

	// PolySymbolExts 多语言词法函数表参与面（与 polyglot.go 私有 polyExts
	// 的键同口径，防漂移由 langcap_test 的等值断言钉住）。声明面在此；
	// polyExts 的 ext→scanner 映射是执行面。
	PolySymbolExts = []string{".py", ".pyw", ".js", ".mjs", ".cjs", ".ts", ".tsx", ".jsx", ".sh", ".bash"}

	// PreciseEdgeLangs 类型解析级（typed）调用边的覆盖语言。Go 走
	// go/types typeutil.Callee 唯一路径（精确边比词法面无同名假边）；
	// 其它语言暂无 typed backend，词法面兜底。
	PreciseEdgeLangs = []string{"Go"}
)

// extLang 扩展名 → 语言展示名（描述口径派生用）。
var extLang = map[string]string{
	".go": "Go", ".py": "Python", ".pyw": "Python",
	".js": "JavaScript", ".mjs": "JavaScript", ".cjs": "JavaScript", ".jsx": "JavaScript",
	".ts": "TypeScript", ".tsx": "TypeScript",
	".sh": "shell", ".bash": "shell", ".sql": "SQL", ".rs": "Rust",
	".java": "Java", ".kt": "Kotlin",
	".c": "C/C++", ".h": "C/C++", ".cpp": "C/C++", ".hpp": "C/C++", ".cc": "C/C++", ".cxx": "C/C++",
	".rb": "Ruby", ".php": "PHP", ".cs": "C#", ".swift": "Swift", ".scala": "Scala",
	".vue": "Vue", ".proto": "protobuf",
	".md": "Markdown", ".yaml": "YAML", ".yml": "YAML", ".toml": "TOML",
}

// HasExt ext（含点）是否在集内。
func HasExt(set []string, ext string) bool {
	for _, e := range set {
		if e == ext {
			return true
		}
	}
	return false
}

// JoinLangs 集内扩展名对应语言展示名的斜杠拼接（工具描述口径派生）。
func JoinLangs(set []string) string { return strings.Join(LangsOf(set), "/") }

// LangsOf 集内扩展名对应的去重语言展示名（保持集内首见顺序，确定性输出）。
// 未知扩展名按原样透出（新扩展名入集未登记展示名时描述面不静默丢）。
func LangsOf(set []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(set))
	for _, e := range set {
		name, ok := extLang[e]
		if !ok {
			name = e
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// concatExts 族并集（保序去重）。
func concatExts(sets ...[]string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 8)
	for _, s := range sets {
		for _, e := range s {
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}
