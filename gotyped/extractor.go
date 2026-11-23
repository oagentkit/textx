// Package gotyped —— go/types 精确调用边抽取器（oagentkit/textx#106）。
//
// codeintel 词法面（go/ast 文本定性）对同名函数/跨包调用/方法值分派全判
// 不准（同名假边）；本包经 golang.org/x/tools/go/packages 全量类型解析 +
// typeutil.Callee types 级定位被调对象，产出精确 caller→callee 边。Go-only；
// 其它语言继续走词法面，由消费方按能力混合出图。
//
// 为什么是本仓的嵌套模块：go/packages 重依赖（x/mod/x/sync 等）会抬 textx
// 根模块（tier 0，只依赖 oekit/pkg）的地板——重活依赖按矩阵纪律隔离在
// 嵌套模块（registry/nacosimpl 同款先例）。消费方按需引入，零 cgo。
//
// 抽取口径：pattern "./..." 装载主模块，带 Body 的 FuncDecl 视为 caller；
// body 内 CallExpr 经 typeutil.Callee 定位被调对象；端点为结构化 Symbol
// （Dir=仓库相对目录/Recv=接收者基类型名/Name），键格式化留给消费方。
//
// 边界（如实声明）：
//   - 接口方法调用解析到接口方法声明（types 级事实）——具体实现间的边
//     不建，宁缺勿假边；
//   - 函数字面量内的调用归外层 FuncDecl；包级函数字面量无外层声明不产边；
//   - 测试文件（_test.go）不在装载面，测试专属调用边不可见；
//   - stdlib/外部依赖剪掉，不建边——图面只表达仓库内符号。
//
// 失败语义：go.mod 缺失/无 module 声明、Load 失败、任一仓内包带 Errors
// （部分解析的边集是不完整的精确事实，宁可失败也不交半成品）、Go 文件
// 总数超 MaxFiles（DefaultMaxFiles=3000），一律返回 error 由调用方降级
// 词法面。
package gotyped

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// DefaultMaxFiles Go 源文件总数上限（./... 装载面内的非测试文件）：全量
// go/types 解析的成本与文件数线性相关，超大仓一次解析秒级起步——超限视为
// 成本不可承受，返回 error 走词法降级而非拖垮调用方（宁降级不可失控）。
const DefaultMaxFiles = 3000

// Symbol 边端点（仓库内符号的三段结构化身份）。
type Symbol struct {
	Dir  string // 仓库相对目录（module 前缀已剥；根包空串）
	Recv string // 接收者基类型名（函数为空串；指针/泛型实例化剥到 Named 基名）
	Name string // 函数/方法名
}

// Edge 一条精确调用边（From 调用 To）。
type Edge struct {
	From Symbol
	To   Symbol
}

// Extractor 精确调用边抽取器（零外部状态：无缓存无配置无日志，每次 Edges
// 全量解析；去重与失败闩留在调用方，本结构保持纯净可复用）。
type Extractor struct {
	// MaxFiles Go 源文件总数上限（超限返回 error）。零/负值按
	// DefaultMaxFiles 处理（测试注入小值验证超限路径）。
	MaxFiles int
}

// New 构建缺省上限的抽取器。
func New() *Extractor {
	return &Extractor{MaxFiles: DefaultMaxFiles}
}

// limit 生效上限（零值规范）。
func (x *Extractor) limit() int {
	if x.MaxFiles > 0 {
		return x.MaxFiles
	}
	return DefaultMaxFiles
}

// Edges 返回全仓精确调用边（有序去重，From/To 字典序）。dir 须为含
// go.mod 的仓库根。失败语义见包文档。
func (x *Extractor) Edges(ctx context.Context, dir string) ([]Edge, error) {
	mod, err := modulePath(dir)
	if err != nil {
		return nil, err
	}
	cfg := &packages.Config{
		Mode:    packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedModule,
		Dir:     dir,
		Context: ctx,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("gotyped: load %s: %w", dir, err)
	}
	// 仓内包过滤：PkgPath 剥 module 前缀成功才算仓内；任一仓内包带
	// Errors 即整体失败——部分解析的边集是不完整的精确事实。
	type inPkg struct {
		pkg *packages.Package
		dir string
	}
	var inmod []inPkg
	files := 0
	for _, p := range pkgs {
		d, ok := inModule(p.PkgPath, mod)
		if !ok {
			continue
		}
		if len(p.Errors) > 0 {
			return nil, fmt.Errorf("gotyped: 包 %s 带错误: %v", p.PkgPath, p.Errors[0])
		}
		inmod = append(inmod, inPkg{p, d})
		files += len(p.Syntax)
	}
	if len(inmod) == 0 {
		return nil, fmt.Errorf("gotyped: %s 无仓内 Go 包", dir)
	}
	if files > x.limit() {
		return nil, fmt.Errorf("gotyped: Go 文件 %d 超上限 %d（%s）", files, x.limit(), dir)
	}

	var out []Edge
	seen := map[Edge]bool{}
	for _, ip := range inmod {
		for _, f := range ip.pkg.Syntax {
			for _, decl := range f.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue // 非 FuncDecl / 无体外声明不产边
				}
				caller := Symbol{Dir: ip.dir, Recv: recvBaseDecl(fd.Recv), Name: fd.Name.Name}
				// ast.Inspect 覆盖嵌套函数字面量——字面量内调用归外层 FuncDecl。
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					edge, ok := calleeEdge(ip.pkg.TypesInfo, call, caller, mod)
					if !ok || seen[edge] {
						return true
					}
					seen[edge] = true
					out = append(out, edge)
					return true
				})
			}
		}
	}
	slices.SortFunc(out, func(a, b Edge) int {
		if c := strings.Compare(a.From.String(), b.From.String()); c != 0 {
			return c
		}
		return strings.Compare(a.To.String(), b.To.String())
	})
	return out, nil
}

// String 端点键的稳定文本形态（Dir.Recv.Name / Dir.Name；根包 Dir 省略点）。
func (s Symbol) String() string {
	base := s.Name
	if s.Recv != "" {
		base = s.Recv + "." + s.Name
	}
	if s.Dir == "" {
		return base
	}
	return s.Dir + "." + base
}

// calleeEdge 单个 CallExpr → 精确边（types 级解析）：typeutil.Callee 定位
// 被调对象；非 *types.Func（内建/类型转换/函数值变量调用）与 stdlib/外部
// 被调剪掉不建边。
func calleeEdge(info *types.Info, call *ast.CallExpr, caller Symbol, mod string) (Edge, bool) {
	fn, ok := typeutil.Callee(info, call).(*types.Func)
	if !ok {
		return Edge{}, false
	}
	p := fn.Pkg()
	if p == nil {
		return Edge{}, false // 万能作用域对象（理论上仅内建），防御
	}
	cdir, ok := inModule(p.Path(), mod)
	if !ok {
		return Edge{}, false // stdlib/外部依赖剪掉
	}
	recv := ""
	if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
		recv = recvBaseType(sig.Recv().Type())
	}
	return Edge{From: caller, To: Symbol{Dir: cdir, Recv: recv, Name: fn.Name()}}, true
}

// inModule PkgPath → 仓库相对目录（剥 module 前缀；根包返回空串）。非本
// module 路径（stdlib/外部依赖）返回 false——调用边剪掉的判据。
func inModule(pkgPath, mod string) (string, bool) {
	if pkgPath == mod {
		return "", true
	}
	if rest, ok := strings.CutPrefix(pkgPath, mod+"/"); ok {
		return rest, true
	}
	return "", false
}

// modulePath 读 dir/go.mod 的 module 声明行（go.mod 是文本协议，字段扫描
// 足够——不引 golang.org/x/mod）。缺失或无 module 行返回 error。
func modulePath(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("gotyped: 读 go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i] // 剥行尾注释（module 值不含 "//"，安全）
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("gotyped: %s 无 module 声明", filepath.Join(dir, "go.mod"))
}

// recvBaseDecl 声明侧接收者基类型名：*T / (T) / T[_] / (*T[_]) 等形态剥
// 到 T；无接收者返回空串。AST 侧仅认 Ident 基名——极端形态（点选表达式的
// 非法接收者）返回空串，与词法抽取同款容错。
func recvBaseDecl(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	t := fl.List[0].Type
loop:
	for {
		switch tt := t.(type) {
		case *ast.StarExpr:
			t = tt.X
		case *ast.ParenExpr:
			t = tt.X
		case *ast.IndexExpr:
			t = tt.X // 泛型接收者 List[T]
		case *ast.IndexListExpr:
			t = tt.X // 多类型参数 List[K, V]
		default:
			break loop
		}
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// recvBaseType 类型侧接收者基类型名（callee 端）：*T / 别名 / 泛型实例化
// 剥到 Named 声明名；接口方法（无接收者）不进入本函数——调用方已判空。
func recvBaseType(t types.Type) string {
loop:
	for {
		switch tt := t.(type) {
		case *types.Pointer:
			t = tt.Elem()
		default:
			break loop
		}
	}
	if named, ok := types.Unalias(t).(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}
