package codeintel

// langcap_test.go —— 语言×能力矩阵（#107）：派生面与族并集的确定性、
// 描述口径派生、声明面↔执行面防漂移等值断言。

import "testing"

// TestLangcapDerivedFromFamilies 可由族派生的能力面必须等于族并集（真单源：
// 改族即改面，不两处手抄）。
func TestLangcapDerivedFromFamilies(t *testing.T) {
	wantScan := concatExts(ShellExts, SQLExts, PyExts, JSTSExts, RustExts, JVMExts, CExts)
	assertSameSet(t, "ScanExts", ScanExts, wantScan)

	wantFuncDiff := concatExts(ShellExts, PyExts, RustExts, JSTSExts)
	assertSameSet(t, "FuncDiffExts", FuncDiffExts, wantFuncDiff)
}

// TestLangcapIndependentChunkFace ChunkExts 是刻意独立的语义分块历史面——
// 不与任何族并集 accidentally 相等（若未来有意并族，改本测试并写明理由）。
func TestLangcapIndependentChunkFace(t *testing.T) {
	assertSameSet(t, "ChunkExts", ChunkExts, concatExts(GoExts, []string{
		".md", ".js", ".ts", ".jsx", ".tsx", ".py", ".java", ".rs", ".c", ".cc",
		".cpp", ".h", ".hpp", ".rb", ".php", ".cs", ".kt", ".swift", ".scala",
		".sh", ".sql", ".yaml", ".yml", ".toml", ".proto", ".vue",
	}))
}

// TestPolySymbolExtsMatchesExecution 声明面（PolySymbolExts）与执行面
// （polyglot polyExts 的键）等值——扩面漏一边即红（防漂移门禁）。
func TestPolySymbolExtsMatchesExecution(t *testing.T) {
	execKeys := make([]string, 0, len(polyExts))
	for k := range polyExts {
		execKeys = append(execKeys, k)
	}
	seen := map[string]bool{}
	for _, e := range PolySymbolExts {
		if _, ok := polyExts[e]; !ok {
			t.Fatalf("PolySymbolExts 声明 %q 不在执行面 polyExts 内", e)
		}
		seen[e] = true
	}
	for k := range polyExts {
		if !seen[k] {
			t.Fatalf("执行面 polyExts 的 %q 未在 PolySymbolExts 声明", k)
		}
	}
}

// TestLangsOfJoinLangs 描述口径派生：去重保序、未知扩展名原样透出。
func TestLangsOfJoinLangs(t *testing.T) {
	langs := LangsOf([]string{".go", ".py", ".pyw", ".js", ".unknownext"})
	want := []string{"Go", "Python", "JavaScript", ".unknownext"}
	if len(langs) != len(want) {
		t.Fatalf("LangsOf=%v, want %v", langs, want)
	}
	for i := range want {
		if langs[i] != want[i] {
			t.Fatalf("LangsOf[%d]=%q, want %q（全量 %v）", i, langs[i], want[i], langs)
		}
	}
	if got := JoinLangs([]string{".go", ".py"}); got != "Go/Python" {
		t.Fatalf("JoinLangs=%q, want Go/Python", got)
	}
}

// TestHasExt 成员判定。
func TestHasExt(t *testing.T) {
	if !HasExt(FuncDiffExts, ".sh") || HasExt(FuncDiffExts, ".go") {
		t.Fatalf("FuncDiffExts 成员判定错: %v", FuncDiffExts)
	}
}

// TestPreciseEdgeLangs typed 边覆盖语言声明非空且含 Go。
func TestPreciseEdgeLangs(t *testing.T) {
	if len(PreciseEdgeLangs) == 0 || !HasExt(PreciseEdgeLangs, "Go") {
		t.Fatalf("PreciseEdgeLangs 应含 Go: %v", PreciseEdgeLangs)
	}
}

func assertSameSet(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s 长度 %d != %d\ngot  %v\nwant %v", name, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d]=%q, want %q（全量 got %v / want %v）", name, i, got[i], want[i], got, want)
		}
	}
}
