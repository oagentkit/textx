// Package spill 溢出存档平面单源（批三百零一）：budgetTool 截断落盘（engine/runner）/
//
// 生命周期: Stable（v0.33.0 定级）——只允许 additive 变更（新增导出符号 /
// 新增方法且给零值默认实现 / 新增错误值）；改签名、改语义、改默认值、
// 删导出符号都算 breaking，判定规则与变更流程见 docs/api-compat.md，
// 逐包理由见 api-lifecycle.yaml。
//
// read_spill 工具取回（engine/tools）/控制台取回面（biz/handler）/写侧清理共用同一套
// 文件名净化、会话隔离与分页语义——LLM 面与控制台面各写一份必漂移（会话隔离三重守卫
// 的判据只应存在一处）。纯 stdlib 叶子层（kbgroup 先例）。
//
// 提示文案契约单源：截断提示文本由 TruncHint 生成、ParseHint 解析——前端
// web/src/utils/timeline/mappers.ts spillHintOf 按同一契约解析（双侧钉档：文案变更
// 须同步该文件与其单测）。
package spill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/oekit/pkg/errors"
	"github.com/oekit/pkg/pathx"
)

// 契约锚点（与前端 spillHintOf 同判据）：截断标记「输出超预算已截断」+ 定位符
// 「spill:<name>」（name 限定安全字符集，解析面与净化面同集）。
var hintLocatorRe = regexp.MustCompile(`spill:([A-Za-z0-9._-]+)`)

// TruncNote 参数：省略字节数/全长/存档定位与路径（retrievable=read_spill 已授予）。
type TruncNote struct {
	Omit        int
	Total       int
	Retrievable bool
	Name        string // 存档文件名（spill 定位符）
	Path        string // 服务器路径（retrievable=false 时的落盘提示）
}

// TruncHint 生成截断提示（自 engine/runner/output.go 迁出逐字保契约）。
func TruncHint(n TruncNote) string {
	if n.Name == "" && n.Path == "" {
		return fmt.Sprintf("\n…【输出超预算已截断：省略中段 %d 字节（全长 %d），建议改用工具的过滤/分页参数缩小范围重查】…\n", n.Omit, n.Total)
	}
	if n.Name != "" && n.Retrievable {
		return fmt.Sprintf("\n…【输出超预算已截断：省略中段 %d 字节（全长 %d）；全文已存档 spill:%s——需要中段原文时用 read_spill 工具分页取回（name=%q，offset/limit 续页）；优先仍建议用采集工具的过滤/分页参数缩小范围重查】…\n", n.Omit, n.Total, n.Name, n.Name)
	}
	return fmt.Sprintf("\n…【输出超预算已截断：省略中段 %d 字节（全长 %d）；全文已落盘: %s；建议改用工具的过滤/分页参数缩小范围重查】…\n", n.Omit, n.Total, n.Path)
}

// ParseHint 解析截断提示：返回定位符与截断旗标（无提示=零值）。Go 侧消费者
// （tool_result 事件结构化字段）与前端 spillHintOf 同判据——改契约双侧同步。
func ParseHint(text string) (locator string, truncated bool) {
	if !strings.Contains(text, "输出超预算已截断") {
		return "", false
	}
	if m := hintLocatorRe.FindStringSubmatch(text); len(m) == 2 {
		return m[1], true
	}
	return "", true
}

// Item 会话存档清单条目（控制台取回面列表行）。
type Item struct {
	Name    string
	Bytes   int64
	ModTime time.Time
}

// ErrBadName 定位符非法（含分隔符/点段/空名）。
var ErrBadName = errors.New("定位符非法（只允许本会话存档目录下的裸文件名）")

// sanitizeName 定位符净化（read_spill 三重守卫之一/二）：单段裸文件名，
// 含分隔符/点段即拒——LLM 面与控制台面同一判据。
//
// 判据委托 oekit/pkg/pathx.IsSafeFileName：它是**判定器**（不合规则直接说不合格，
// 不改写），且比手搓版更严——除分隔符与 "."/".." 点段外，还拒 Windows 盘符
// 相对路径（"C:foo.bundle"），fail-closed（同一段载荷在 Linux/Windows 上得到
// 同一结论）。
//
// 旧实现用 filepath.Base 判等，而 filepath.Base("..") == ".."、filepath.Base(".") == "."
// ——两个点段都会被放行；改用 pathx 后一并堵上。
func sanitizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !pathx.IsSafeFileName(name) {
		return "", ErrBadName
	}
	return name, nil
}

// sessionDir 会话目录拼装+前缀守卫（三重守卫之三）：净化后理论出不去，仍显式
// 校验落点（防御纵深，read_spill 原语义）。
//
// 会话标识同样走 pathx.IsSafeFileName：旧实现的 filepath.Base 判等会把 ".."
// 放行（Base("..") == ".."），于是 sessionID=".." 能让落点指到 spill 根目录的
// **父目录**——三重守卫的前两道（定位符净化）根本不参与会话目录的判定，第三道
// 的前缀校验又按同一 base 算，两道一起失效（实测可读出根目录外的文件）。
func sessionDir(root, sessionID string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("spill 面未装配（mcp.spill_dir 空），无存档可取")
	}
	sess := strings.TrimSpace(sessionID)
	if sess == "" {
		return "", errors.New("无法归属会话（缺 session 标识），拒绝执行")
	}
	if !pathx.IsSafeFileName(sess) {
		return "", errors.New("会话标识非法")
	}
	return filepath.Join(root, sess), nil
}

// resolve 会话隔离取文件路径（净化+前缀校验）。
func resolve(root, sessionID, name string) (dir, p string, err error) {
	dir, err = sessionDir(root, sessionID)
	if err != nil {
		return "", "", err
	}
	base, err := sanitizeName(name)
	if err != nil {
		return "", "", err
	}
	p = filepath.Join(dir, base)
	if !strings.HasPrefix(filepath.Clean(p), filepath.Clean(dir)+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("定位符越界（不在本会话存档目录内）: %q", name)
	}
	return dir, p, nil
}

// List 会话存档清单（mtime 降序，最新在前）。
func List(root, sessionID string) ([]Item, error) {
	dir, err := sessionDir(root, sessionID)
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Item{}, nil // 无存档=空清单不是错误
		}
		return nil, err
	}
	items := make([]Item, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		items = append(items, Item{Name: e.Name(), Bytes: info.Size(), ModTime: info.ModTime()})
	}
	slices.SortFunc(items, func(a, b Item) int { return b.ModTime.Compare(a.ModTime) })
	return items, nil
}

// Page 分页取回结果（read_spill 回包与控制台取回面同构）。
type Page struct {
	Name       string `json:"name"`
	Offset     int64  `json:"offset"`
	Bytes      int    `json:"bytes"`
	Total      int64  `json:"total"`
	EOF        bool   `json:"eof"`
	Content    string `json:"content"`
	NextOffset int64  `json:"next_offset,omitempty"`
}

// ReadPage 会话隔离分页取回：off<0 视作 0、>total 收到 total；limit<=0 用缺省，
// 超上限收上限（LLM 面与控制台面同一缺省/上限）。
//
// # 页边界不切坏字符（issue #199）
//
// 分页按字节预算切，但 Page.Content 直接进 read_spill 回包 / 控制台取回面
// （字段带 json 标签，encoding/json 对非法 UTF-8 替换为 U+FFFD）——页边界
// 落在多字节字符中间 = 每个跨页字符丢 1–2 字节且不可复原，与截断提示承诺的
// 「分页取回中段原文」直接冲突。修法三条：
//
//  1. **尾部回退**：预算终点落在字符中间时回退到最近的 rune 起点，被截的
//     字符**整字符挪到下一页**（NextOffset 指向它的起点）——顺序分页逐页
//     拼接与原文逐字节相等，无丢无损；
//  2. **头部回退**：非顺序分页传入的 off 落在字符中间时，回退到所在字符
//     的起点再读（页与页允许重叠，不允许跳字节——任何字节都在某一页里）；
//  3. **进度保底**：limit 小于单字符宽度（回退后整页装不下一个字符）时，
//     至少前进一个完整字符（Bytes 可略超 limit，至多 utf8.UTFMax-1），
//     分页器不死循环。
func ReadPage(root, sessionID, name string, off int64, limit int) (Page, error) {
	_, p, err := resolve(root, sessionID, name)
	if err != nil {
		return Page{}, err
	}
	f, err := os.Open(p)
	if err != nil {
		return Page{}, errors.Wrap(err, "存档不存在或不可读")
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return Page{}, errors.Wrap(err, "存档状态读取失败")
	}
	total := st.Size()
	if off < 0 {
		off = 0
	}
	if off > total {
		off = total
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	// 头部回退：off 落在多字节字符中间（非顺序分页的入参）时退到所在字符
	// 起点。字节 [旧off, 新off) 与上一页重叠——重叠无害，跳字节才是丢失。
	if off > 0 && off < total {
		if back := midRuneBack(f, off); back > 0 {
			off -= back
		}
	}
	n := total - off
	if n <= 0 {
		// 空尾页（off==total，越界 offset 收到 total 的既有口径）。
		return Page{Name: filepath.Base(p), Offset: off, Total: total, EOF: true}, nil
	}
	if n > int64(limit) {
		n = int64(limit)
	}
	// 多读 utf8.UTFMax 字节做判定余量：只为看清「预算终点是不是落在字符
	// 中间」（以及进度保底时找完整字符的终点），多余字节不进 Content。
	buf := make([]byte, n+utf8.UTFMax)
	got, rerr := f.ReadAt(buf, off)
	if rerr != nil && got == 0 {
		return Page{}, errors.Wrap(rerr, "存档读取失败")
	}
	cut := got
	if int64(got) > n {
		// 预算装满且文件还有后续：终点可能切进字符，回退到最近的 rune 起点
		//（被截字符整字符挪到下一页）。got<=n 即已读到 EOF：文件自己的
		// 尾字节问题不属分页切坏，如实带出。
		cut = int(n)
		for cut > 0 && !utf8.RuneStart(buf[cut]) {
			cut--
		}
		if cut == 0 {
			// 进度保底：预算小于单字符宽度，整页装不下一个字符——至少
			// 前进一个完整字符（略超预算 ≤ utf8.UTFMax-1，分页器不死循环）。
			cut = 1
			for cut < got && !utf8.RuneStart(buf[cut]) {
				cut++
			}
		}
	}
	pg := Page{
		Name: filepath.Base(p), Offset: off, Bytes: cut, Total: total,
		EOF: off+int64(cut) >= total, Content: string(buf[:cut]),
	}
	if !pg.EOF {
		pg.NextOffset = off + int64(cut)
	}
	return pg, nil
}

// midRuneBack 探测 off 是否落在多字节字符中间，是则返回应回退的字节数
// （≤ utf8.UTFMax-1；最多回探 3 字节——再往前还找不到起点即文件本身不是
// 合法 UTF-8，如实带出不回退）。只做 1–2 次小 ReadAt，顺序分页（off 恰在
// rune 起点）零回退零额外读。
func midRuneBack(f *os.File, off int64) int64 {
	probe := make([]byte, 1)
	if got, _ := f.ReadAt(probe, off); got == 1 && utf8.RuneStart(probe[0]) {
		return 0 // off 恰在字符起点（顺序分页的常态）
	}
	head := make([]byte, utf8.UTFMax-1)
	base := off - int64(len(head))
	if base < 0 {
		base = 0
		head = head[:off]
	}
	got, _ := f.ReadAt(head, base)
	for i := int64(got) - 1; i >= 0; i-- {
		if utf8.RuneStart(head[i]) {
			return off - (base + i) // 最近（离 off 最近）的字符起点
		}
	}
	return 0
}

// 单次取回字节缺省/上限（自 read_spill 迁出：取回文本直接进上下文，上限与输出
// 预算同数量级——防「分页取回」退化成「全文重塞」）。
const (
	DefaultLimit = 8192
	MaxLimit     = 16384
)

// Prune 写侧清理（best-effort，清理失败不影响本次存档）：keepNewest 数护栏（≤0=不按数）
// + ttl 按龄（0=不按龄）——「按龄清理」挂账的执行体；无后台任务纪律不变（写侧顺手清）。
func Prune(dir string, keepNewest int, ttl time.Duration) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type mf struct {
		name string
		mod  time.Time
	}
	files := make([]mf, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		files = append(files, mf{e.Name(), info.ModTime()})
	}
	slices.SortFunc(files, func(a, b mf) int { return a.mod.Compare(b.mod) }) // 最旧在前

	// 按龄：超龄即删（keepNewest 数护栏另算，不互相豁免——量纲不同各管各）
	if ttl > 0 {
		cut := time.Now().Add(-ttl)
		kept := files[:0]
		for _, f := range files {
			if f.mod.Before(cut) {
				_ = os.Remove(filepath.Join(dir, f.name))
				continue
			}
			kept = append(kept, f)
		}
		files = kept
	}
	// 按数：保留最新 keepNewest 份
	if keepNewest > 0 && len(files) > keepNewest {
		for _, f := range files[:len(files)-keepNewest] {
			_ = os.Remove(filepath.Join(dir, f.name))
		}
	}
}
