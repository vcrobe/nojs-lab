//go:build js || wasm
// +build js wasm

package main

import (
	"syscall/js"

	"github.com/vcrobe/nojs/console"
	"github.com/vcrobe/nojs/dialogs"
)

func add(this js.Value, args []js.Value) interface{} {
	a := args[0].Int()
	b := args[1].Int()
	return js.ValueOf(a + b)
}

func showLogs() {
	console.Log("this is a log", 5+2, 3.2)
	console.Warning("this is a warning")
	console.Error("this is an error")
}

func showAlert() {
	dialogs.Alert("This is my alert")
}

func showPrompt() {
	name := dialogs.Prompt("write your name")

	if name == "<null>" {
		println("you pressed the cancel button")
	} else if name == "" {
		println("the string is empty")
	} else {
		println("your name is", name)
	}
}

func callJsFunction() {
	js.Global().Call("calledFromGoWasm", "Hello from Go!")
}

func main() {
	// Export the `add` function to JavaScript
	js.Global().Set("add", js.FuncOf(add))

	// Call the JavaScript function
	callJsFunction()

	// Keep the Go program running
	select {}
}
