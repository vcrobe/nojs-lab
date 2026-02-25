//go:build js || wasm

package components

import (
	"github.com/ForgeLogic/nojs/runtime"
)

// Counter demonstrates reactive state via StateHasChanged().
type Counter struct {
	runtime.ComponentBase

	Count int
}

func (c *Counter) OnMount() {
	println("Counter mounted")
}

func (c *Counter) OnParametersSet() {
	println("OnParametersSet: ", c.Count)
}

func (c *Counter) OnUnmount() {
	println("Counter unmounted")
}

func (c *Counter) Increment() {
	c.Count++
	c.StateHasChanged()
	
	println("Incremented: ", c.Count)
}
