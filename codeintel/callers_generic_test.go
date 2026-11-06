package codeintel

import (
	"path/filepath"
	"testing"
)

// genericCallerFixture 四个方法**同名** Wrap、只有接收者形态不同，
// 各自的调用位都指向同一个 Target。这是 Caller 归属的最小判别场：
// 谁的接收者被丢了，谁的 Caller 就退化成裸 "Wrap"，彼此不可区分。
//
//	*Repo[T]        单类型参数、指针接收者   ⇒ *ast.IndexExpr
//	*Pair[K, V]     多类型参数、指针接收者   ⇒ *ast.IndexListExpr
//	Val[T]          单类型参数、值接收者     ⇒ IndexExpr 无 StarExpr 包裹
//	*Plain          非泛型对照               ⇒ 既有路径不许回归
func genericCallerFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/gx\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, "gen.go"), `package gx

// Target 被测调用目标。
func Target() {}

// Repo 单类型参数的泛型接收者。
type Repo[T any] struct{ v T }

// Wrap 泛型接收者上的方法。
func (r *Repo[T]) Wrap(x T) { Target() }

// Pair 多类型参数的泛型接收者。
type Pair[K comparable, V any] struct {
	k K
	v V
}

// Wrap 多类型参数接收者上的同名方法。
func (p *Pair[K, V]) Wrap(x V) { Target() }

// Val 值接收者的泛型类型。
type Val[T any] struct{ v T }

// Wrap 值接收者上的同名方法。
func (v Val[T]) Wrap(x T) { Target() }

// Plain 非泛型对照。
type Plain struct{}

// Wrap 非泛型接收者上的同名方法。
func (p *Plain) Wrap(x int) { Target() }
`)
	return dir
}

// TestPreciseCallersGenericReceiver 泛型接收者的 Caller 必须带类型名。
func TestPreciseCallersGenericReceiver(t *testing.T) {
	x, err := NewIndex(genericCallerFixture(t), 100)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	refs := x.PreciseCallers("Target", 0)
	if len(refs) == 0 {
		t.Fatal("无命中")
	}

	got := map[string]int{}
	for _, r := range refs {
		got[r.Caller]++
	}

	want := []string{"Repo.Wrap", "Pair.Wrap", "Val.Wrap", "Plain.Wrap"}
	for _, w := range want {
		switch got[w] {
		case 1:
		case 0:
			t.Errorf("缺 Caller=%q（实得 %v）", w, got)
		default:
			t.Errorf("Caller=%q 出现 %d 次，应为 1（实得 %v）", w, got[w], got)
		}
	}
	// 裸方法名即「接收者被丢弃」的具体形态：一旦出现，上面四条同名方法就不可区分。
	if n, ok := got["Wrap"]; ok {
		t.Errorf("出现裸 Caller=\"Wrap\" %d 次——接收者类型名被丢弃（实得 %v）", n, got)
	}
	if len(got) != len(want) {
		t.Errorf("Caller 去重后 %d 种，应为 %d 种：%v", len(got), len(want), got)
	}
}

// TestPreciseCallersGenericReceiverCallee 泛型方法体内的方法调用照常解析：
// 补 IndexExpr 分支只该补接收者归属，不该让 Callee 一起退化。
func TestPreciseCallersGenericReceiverCallee(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/gy\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, "h.go"), `package gy

// Save 被测目标方法。
func (r *Repo[T]) Save() {}

// Repo 泛型接收者。
type Repo[T any] struct{ v T }

// Wrap 在泛型方法体内调用 Save。
func (r *Repo[T]) Wrap() { r.Save() }
`)
	x, err := NewIndex(dir, 100)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	refs := x.PreciseCallers("Save", 0)
	if len(refs) != 1 {
		t.Fatalf("命中 %d 条，应为 1 条：%+v", len(refs), refs)
	}
	if refs[0].Caller != "Repo.Wrap" {
		t.Errorf("Caller=%q，应为 \"Repo.Wrap\"", refs[0].Caller)
	}
	if refs[0].Callee != "r.Save" {
		t.Errorf("Callee=%q，应为 \"r.Save\"", refs[0].Callee)
	}
}
