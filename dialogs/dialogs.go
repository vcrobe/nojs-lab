//go:build js || wasm

package dialogs

import (
	"syscall/js"
)

func Alert(msg string) {
	js.Global().Call("alert", msg)
}

func Prompt(message string) string {
	result := js.Global().Call("prompt", message)
	return result.String()
}
