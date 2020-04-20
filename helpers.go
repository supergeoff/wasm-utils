// +build js, wasm

package utils

import "syscall/js"

func Bool(b bool) *bool {
	return &b
}

func ConsoleLog(i interface{}) {
	v := JsValueOf(i)
	js.Global().Get("console").Call("log", v)
}
