// polyglot.go —— 多语言词法符号表（JS/TS/Python/Shell）。
//
// 边界如实声明：JS/TS/Python/Shell 与 Go 的 get_callers 全部是**词法级**索引
// （正则锚定声明/调用形态），不做类型解析——与 Go 侧 go/scanner 同级质量，
// 零新增依赖、零 cgo（CGO_ENABLED=0 兼容）。误报形态（字符串/注释内命中、
// 动态调用）与 Go 词法索引一致，由工具描述向 agent 声明"须回 get_file 核实"。
package codeintel

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oagentkit/textx/textutil"
)

var pyFuncRe = regexp.MustCompile(`(?m)^(\s*)(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`)

// jsFuncRe JS/TS 函数声明/赋值/导出（词法形态：function、const/let = (…) =>、
// class 方法不做——方法名与调用形态歧义大，宁缺勿滥）。
var jsFuncRe = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?(?:function\s+([A-Za-z_$][A-Za-z0-9_$]*)|(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)(?::[^=
]+)?\s*=\s*(?:async\s*)?(?:\(|[A-Za-z_$][A-Za-z0-9_$]*\s*=>))`)

// shellFuncRe shell 词法函数定义（oagentkit/textx#83）：组 2=POSIX `name() {`
// 形态名（兼容 `function name() {` 混合形态），组 4=`function name {` 形态名。
// 括号内空白用 [ \t] 不用 \s——\s 含 \n 会让跨行残片误命中；形态口径与
// argus internal/langcap 的 shell fn 正则对齐。
var shellFuncRe = regexp.MustCompile(`(?m)^[ \t]*(?:function[ \t]+)?([A-Za-z_][A-Za-z0-9_]*)[ \t]*\([ \t]*\)[ \t\r\n]*\{|^[ \t]*function[ \t]+([A-Za-z_][A-Za-z0-9_]*)[ \t\r\n]*\{`)

// PolySymbol 非 Go 语言的词法符号。
type PolySymbol struct {
	Name string
	Lang string // py | js | shell
	File string // 相对索引根
	Line int
	Args string // 参数表原文（≤120 runes；shell 无参数表，空）
	Doc  string // 前置 docstring/注释首行（≤120 runes，可空）
}

// polyExts 扩展名 → 语言归族（oagentkit/textx#83：.mjs/.cjs 归 js、.pyw 归 py、
// .sh/.bash 归 shell）。
var polyExts = map[string]string{
	".py":   "py",
	".pyw":  "py",
	".js":   "js",
	".mjs":  "js",
	".cjs":  "js",
	".ts":   "js",
	".tsx":  "js",
	".jsx":  "js",
	".sh":   "shell",
	".bash": "shell",
}

// polyScanners 语言 → 词法扫描器。
var polyScanners = map[string]func(text, lang string) []PolySymbol{
	"py":    scanPy,
	"js":    scanJS,
	"shell": scanShell,
}

// scanPolyFiles 扫描 dir 下 py 族（.py/.pyw）、js 族（.js/.mjs/.cjs/.ts/.tsx/.jsx）、
// shell 族（.sh/.bash）文件建词法符号表（上限与 Go 共享
// maxFiles 语义由调用方控制；这里独立 cap 防大仓重复遍历）。
func scanPolyFiles(dir string, maxFiles int) (map[string][]PolySymbol, error) {
	out := map[string][]PolySymbol{}
	if maxFiles <= 0 {
		maxFiles = 1000
	}
	count := 0
	var walkErr error
	werr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if path == dir {
				return err // 根不可达上抛（第八轮审计，与 Go 侧同纪律）
			}
			return nil
		}
		if d.IsDir() {
			if path != dir && skipDirs[strings.ToLower(d.Name())] {
				return filepath.SkipDir
			}
			return nil
		}
		if count >= maxFiles {
			return filepath.SkipAll
		}
		ext := filepath.Ext(path)
		lang, ok := polyExts[ext]
		if !ok {
			return nil
		}
		fi, ierr := d.Info()
		if ierr != nil || fi.Size() > maxPolyFileBytes {
			return nil // stat 失败 fail-closed（防 TOCTOU 无界读；不计 count）
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(dir, path)
		if rerr != nil {
			return nil
		}
		count++
		text := string(src)
		syms := polyScanners[lang](text, lang)
		for i := range syms {
			syms[i].File = filepath.ToSlash(rel)
		}
		for _, s := range syms {
			key := strings.ToLower(s.Name)
			out[key] = append(out[key], s)
		}
		return nil
	})
	if werr != nil {
		walkErr = werr
	}
	return out, walkErr
}

// scanPy Python def（含 docstring 首行启发：def 后首个非空行若为三引号/注释）。
func scanPy(text, lang string) []PolySymbol {
	var out []PolySymbol
	for _, m := range pyFuncRe.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[4]:m[5]]
		args := textutil.TruncEllipsis(strings.TrimSpace(text[m[6]:m[7]]), 120)
		// Line 从名字捕获组（m[4]）起点计数（oagentkit/textx#71）：m[0] 是含
		// 前置空白的匹配起点，Go regexp 的 \s 含 \n——^(\s*) 的最左匹配会在
		// def 前的空行串落锚，text[:m[0]] 数的是空白串之前的换行数 ⇒ Line
		// 偏小量 = 前置空行数。名字组不吃前置空白，计数起点即定义行。
		line := 1 + strings.Count(text[:m[4]], "\n")
		doc := ""
		if nl := strings.Index(text[m[1]:], "\n"); nl >= 0 {
			// def 头匹配末（m[1]）的下一**物理**行——此前对整段剩余文本
			// TrimSpace 会跨空行塌缩，且无条件取首行（"pass" 级正文也被当
			// 文档；多行参数表取到参数名；第八轮审计）。仅三引号/注释形态提取。
			first := strings.TrimSpace(strings.SplitN(text[m[1]+nl+1:], "\n", 2)[0])
			doc = extractDocLine(first)
		}
		out = append(out, PolySymbol{Name: name, Lang: lang, Line: line, Args: args, Doc: doc})
	}
	return out
}

// scanJS JS/TS 词法函数声明（组 2=function 名，组 4=const/let/var 赋值名）。
func scanJS(text, lang string) []PolySymbol {
	var out []PolySymbol
	for _, m := range jsFuncRe.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[2], m[3]
		if s < 0 {
			s, e = m[4], m[5]
		}
		if s < 0 {
			continue
		}
		// Line 从名字捕获组起点计数（oagentkit/textx#71）——m[0] 含前置空白，
		// ^\s* 跨行吃空行后计数偏小；scanPy 同修，两侧对称。
		line := 1 + strings.Count(text[:s], "\n")
		out = append(out, PolySymbol{Name: text[s:e], Lang: lang, Line: line})
	}
	return out
}

// scanShell shell 词法函数定义（oagentkit/textx#83）：组 2=POSIX 名
// （`name() {`，`function name() {` 混合形态同组），组 4=`function name {` 名。
// 与 scanJS 同为名字+行号最小面（shell 词法无参数表；注释内命中按词法口径
// 如实计入，工具描述声明须核实）。
func scanShell(text, lang string) []PolySymbol {
	var out []PolySymbol
	for _, m := range shellFuncRe.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[2], m[3]
		if s < 0 {
			s, e = m[4], m[5]
		}
		if s < 0 {
			continue
		}
		// Line 从名字捕获组起点计数（oagentkit/textx#71），三扫描器同口径。
		line := 1 + strings.Count(text[:s], "\n")
		out = append(out, PolySymbol{Name: text[s:e], Lang: lang, Line: line})
	}
	return out
}

// PolyDefinitions 按名查非 Go 符号（大小写不敏感；Go 侧命中优先——同名冲突
// 时 Go 索引是权威）。
func (x *Index) PolyDefinitions(name string) []PolySymbol {
	if x == nil {
		return nil
	}
	return x.poly[strings.ToLower(name)]
}

// GoCallers 按名查调用形态出现位置（pkg.F( 与 F( 两种形态，定义行不计；
// 与 References 的差异：只收调用形态行，并注明词法级）。v0.12.0 起查建索引
// 期与 refs 同遍采集的 calls 表（token 对 IDENT+'('）——正则扫描、正则缓存、
// 子串预筛、逐行匹配四层全部删除：token 扫描无视空白（Serve (w) 命中）、
// 字符串/注释不产 token（假命中根除）、查询 O(1)。
func (x *Index) GoCallers(name string, limit int) []Ref {
	if x == nil || name == "" {
		return nil
	}
	if limit <= 0 {
		limit = 50
	}
	defLines := map[int64]bool{}
	for _, ds := range x.defs[strings.ToLower(name)] {
		if fid, ok := x.fileIDOf(ds.File); ok {
			defLines[defLineKey(fid, int32(ds.Line))] = true
		}
	}
	var out []Ref
	for _, r := range x.calls[name] {
		if defLines[defLineKey(r.file, r.line)] {
			continue
		}
		out = append(out, Ref{File: x.fileNames[r.file], Line: int(r.line)})
		if len(out) >= limit {
			break
		}
	}
	return out
}

// extractDocLine 文档首行提取：仅三引号 docstring/注释形态生效，
// 剥离定界符后截 120 rune（第八轮审计：此前无条件取首个非空行，
// "pass" 级正文与多行参数表的参数名都会被当文档）。
func extractDocLine(first string) string {
	q3d := string([]byte{'"', '"', '"'})
	q3s := string([]byte{'\'', '\'', '\''})
	if strings.HasPrefix(first, q3d) || strings.HasPrefix(first, q3s) || strings.HasPrefix(first, "#") {
		return textutil.TruncEllipsis(strings.Trim(first, "\"'# \t"), 120)
	}
	return ""
}

// maxPolyFileBytes poly 侧单文件字节上限（词法正则扫描成本高于 AST 解析）。
const maxPolyFileBytes = 2 << 20
