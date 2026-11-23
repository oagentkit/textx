package main

import (
	"example.com/probe/lib"
	"example.com/probe/util"
)

func main() {
	lib.Greet()
	lib.Helper() // 与 util.Helper 同名跨包——词法面会混，typed 边必须分清
	util.Helper()
	w := lib.Widget{}
	w.Run()
}
