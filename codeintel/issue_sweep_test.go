package codeintel

// 2026-11-11 issue 清零战回归测试（#71）：钉住 polyglot 符号行号定位。

import "testing"

// TestPolyLineCountsFromNameGroup_Issue71 def/const 前有空行（常态排版）时
// Line 必须是定义的物理行——^(\s*) 跨行吃掉前置空行后从匹配起点计数曾让
// Line 偏小量 = 前置空行数，get_definitions 指到定义上方。
func TestPolyLineCountsFromNameGroup_Issue71(t *testing.T) {
	pyText := "def a():\n    pass\n\n\ndef b():"
	syms := scanPy(pyText, "py")
	if len(syms) != 2 {
		t.Fatalf("应扫出 2 个符号，实得 %d", len(syms))
	}
	if syms[0].Line != 1 {
		t.Errorf("def a 应在第 1 行，实得 %d", syms[0].Line)
	}
	if syms[1].Line != 5 {
		t.Errorf("def b 应在第 5 行（#71 修复前报 3），实得 %d", syms[1].Line)
	}

	jsText := "const a = () => {};\n\n\nconst foo = () => {};"
	jsyms := scanJS(jsText, "js")
	if len(jsyms) != 2 {
		t.Fatalf("应扫出 2 个 JS 符号，实得 %d", len(jsyms))
	}
	if jsyms[0].Line != 1 {
		t.Errorf("const a 应在第 1 行，实得 %d", jsyms[0].Line)
	}
	if jsyms[1].Line != 4 {
		t.Errorf("const foo 应在第 4 行（#71 修复前报 2），实得 %d", jsyms[1].Line)
	}

	// 带缩进的 def：空白串不跨行时行为不变（回归护栏）。
	indented := "class C:\n    def m(self):\n        pass"
	msyms := scanPy(indented, "py")
	if len(msyms) != 1 || msyms[0].Line != 2 {
		t.Fatalf("缩进 def 行号漂移: %+v", msyms)
	}
}
