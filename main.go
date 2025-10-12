//go:build js || wasm
// +build js wasm

package main

import (
	"github.com/vcrobe/nojs/vdom"
)

func main() {
	// Create and render the test component

	vdom.RenderToSelector("#app", render())

	testComp1 := NewTestComponent("First component", 1)
	vdom.RenderToSelector("#app", testComp1.Render())

	testComp2 := NewTestComponent("Second component", 8)
	vdom.RenderToSelector("#app", testComp2.Render())

	// Keep the Go program running
	select {}
}
