package diffx

import "testing"

// TestHunkLinesMinusHeaderResetsOwner `--- ` 头行必须重置文件归属（textx#31）。
//
// 包注释声明「+++ / --- / diff --git 行在任意位置都重置文件归属」，实现却只
// 处理 diff --git（重置）与 +++（设归属），**从不看 ---**。截断/拼接形态下
// 某块只剩 `--- a/other.go` 头（+++ 被截掉）时，其后的 hunk 会归属到上一个
// 文件——消费方按行区间做注入/定位就打错文件，而且没有任何信号。
func TestHunkLinesMinusHeaderResetsOwner(t *testing.T) {
	// a.go 正常一块，随后是只剩 `---` 头的截断块。
	truncated := "+++ b/a.go\n@@ -1 +1 @@\n+x\n--- a/other.go\n@@ -10,2 +10,2 @@\n-x\n+y\n"
	m := HunkLines(truncated)

	if got := m["a.go"]; len(got) != 1 || got[0] != (LineRange{1, 1}) {
		t.Fatalf("a.go 只该有原始那一个区间，实得 %v——截断块的 hunk 被误归属过来了", got)
	}
	if _, ok := m["other.go"]; ok {
		t.Fatalf("other.go 只有 --- 头（+++ 被截掉），不该入表: %v", m["other.go"])
	}
	if len(m) != 1 {
		t.Fatalf("只该有 a.go 一项，实得 %v", m)
	}
}

// TestHunkLinesMinusHeaderNormalForms `---` 参与重置不得破坏正常形态。
func TestHunkLinesMinusHeaderNormalForms(t *testing.T) {
	// 完整块：--- 清空、+++ 设归属。
	full := "diff --git a/b.go b/b.go\n--- a/b.go\n+++ b/b.go\n@@ -5,2 +5,3 @@\n ctx\n+x\n"
	if got := HunkLines(full)["b.go"]; len(got) != 1 || got[0] != (LineRange{5, 7}) {
		t.Fatalf("完整块的归属错: %v", got)
	}

	// 新增文件：--- /dev/null 清空后由 +++ 设归属，照常入表。
	added := "diff --git a/new.go b/new.go\n--- /dev/null\n+++ b/new.go\n@@ -0,0 +1,2 @@\n+a\n+b\n"
	if got := HunkLines(added)["new.go"]; len(got) != 1 || got[0] != (LineRange{1, 2}) {
		t.Fatalf("新增文件归属错: %v", got)
	}

	// 删除文件：+++ /dev/null 不设归属 ⇒ 无区间（与既有语义一致）。
	removed := "diff --git a/gone.go b/gone.go\n--- a/gone.go\n+++ /dev/null\n@@ -1,3 +0,0 @@\n-a\n-b\n-c\n"
	if got, ok := HunkLines(removed)["gone.go"]; ok {
		t.Fatalf("删除文件新侧无行，不该入表: %v", got)
	}
}
