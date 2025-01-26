//go:build js || wasm

package console

import (
	"syscall/js"
)

func Log(args ...interface{}) {
	console := js.Global().Get("console")
	console.Call("log", args...)
}

func Warning(args ...interface{}) {
	console := js.Global().Get("console")
	console.Call("warn", args...)
}

func Error(args ...interface{}) {
	console := js.Global().Get("console")
	console.Call("error", args...)
}
