// callers.go —— Go 调用方精化：token 级调用位（calls 表，IDENT 紧跟 '('）
// 作候选，AST 二次核实并补封闭函数名与限定名。零磁盘 IO——候选文件源码在
// 建索引期已驻留（fileEntry.src），精化只做内存重解析。
//
// 边界如实声明（与 GoCallers 词法级同纪律）：不做类型解析——同名异接收者的
// 方法调用全部命中（pkg.F 与 var.F 形态在 AST 上不可分）；接口分派/动态派发
// 不可见；包级 var/const 初始化器与匿名函数体内的调用封闭函数记
// "(package init)"。误报面显著小于纯词法（字符串/注释在候选层已排除，本层
// 再排除非调用位），但**不是** SSA 精确调用图。
package codeintel

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// CallerRef 精化后的调用位。
type CallerRef struct {
	File   string // 相对索引根路径
	Line   int
	Caller string // 封闭函数（Recv.Name / Name；包级初始化记 "(package init)"）
	Callee string // 调用目标限定名（pkg.F / T.M / F；链式调用取末端 Sel）
}

// CallerInit 包级初始化/匿名函数体的封闭函数占位。
//
// 导出成常量供下游判别（下游按 `Caller == codeintel.CallerInit` 判断
// 「这条调用位在包级初始化里」），而不是让下游去比一个字面量。
const CallerInit = "(package init)"

// PreciseCallers 按名查精化调用位：与 GoCallers 同一候选源（calls 表，定义行
// 不计），候选文件逐一内存重解析为 AST，只保留「确为调用位」的行并补 Caller/
// Callee。结果按 (File,Line) 稳定有序，limit 封顶（<=0 = 50）。文件解析失败
// 丢弃该文件候选（宁缺毋假——不回填 token 级结果）。查询侧并发安全（与
// Index 只读纪律一致：不写任何 Index 字段）。
func (x *Index) PreciseCallers(name string, limit int) []CallerRef {
	if x == nil || name == "" {
		return nil
	}
	if limit <= 0 {
		limit = 50
	}
	defLines := map[int32]bool{}
	for _, ds := range x.defs[strings.ToLower(name)] {
		if fid, ok := x.fileIDOf(ds.File); ok {
			defLines[fid<<20|int32(ds.Line)] = true
		}
	}
	// 候选按文件聚合（同文件一次解析）；定义行与 GoCallers 同口径剔除
	byFile := map[string][]int{}
	for _, r := range x.calls[name] {
		if defLines[r.file<<20|r.line] {
			continue
		}
		byFile[x.fileNames[r.file]] = append(byFile[x.fileNames[r.file]], int(r.line))
	}
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	fset := token.NewFileSet()
	var out []CallerRef
	for _, rel := range files {
		src, ok := x.fileSrc(rel)
		if !ok {
			continue
		}
		af, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		lines := map[int]bool{}
		for _, l := range byFile[rel] {
			lines[l] = true
		}
		out = append(out, declCalls(fset, af, rel, lines)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// fileSrc 相对路径 → 建索引期驻留的源码（未驻留返回 false）。
func (x *Index) fileSrc(rel string) ([]byte, bool) {
	for i := range x.files {
		if x.files[i].rel == rel {
			return x.files[i].src, true
		}
	}
	return nil, false
}

// declCalls 遍历顶层声明收集候选行上的调用位：FuncDecl 的 Caller 为
// Recv.Name / Name；其余声明（var/const/import 初始化表达式）记 CallerInit。
func declCalls(fset *token.FileSet, af *ast.File, rel string, lines map[int]bool) []CallerRef {
	var out []CallerRef
	collect := func(caller string, body ast.Node) {
		ast.Inspect(body, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok || ce.Lparen == token.NoPos {
				return true
			}
			line := fset.Position(ce.Lparen).Line
			if !lines[line] {
				return true
			}
			callee := calleeName(ce.Fun)
			if callee == "" {
				return true
			}
			out = append(out, CallerRef{File: rel, Line: line, Caller: caller, Callee: callee})
			return true
		})
	}
	for _, decl := range af.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			collect(funcDeclName(fd), fd)
			continue
		}
		collect(CallerInit, decl)
	}
	return out
}

// funcDeclName 函数声明展示名（Recv.Name / Name）。
func funcDeclName(fd *ast.FuncDecl) string {
	name := fd.Name.Name
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		if recv := recvTypeName(fd.Recv.List[0].Type); recv != "" {
			name = recv + "." + name
		}
	}
	return name
}

// recvTypeName 接收者类型名（*T / T / pkg.T → T / pkg.T）。
func recvTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return recvTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

// calleeName 调用目标限定名：Ident → F；pkg/T/var 的 SelectorExpr → X.Sel；
// 链式调用（f().g）类型不可知只取末端 Sel；其余（索引表达式等）返回空。
func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		sel := f.Sel.Name
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name + "." + sel
		}
		return sel
	default:
		return ""
	}
}
