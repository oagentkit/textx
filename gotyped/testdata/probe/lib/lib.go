package lib

import "example.com/probe/util"

// Greet 调用 util.Helper。
func Greet() {
	util.Helper()
}

// Helper 与 util.Helper 同名（跨包同名函数——词法抽边的经典假边源）。
func Helper() {}

// Widget 值接收者方法（端点 Recv 判定用）。
type Widget struct{}

// Run 调用 util.Helper。
func (w Widget) Run() {
	util.Helper()
}
