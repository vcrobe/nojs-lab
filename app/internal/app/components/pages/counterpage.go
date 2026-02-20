//go:build js || wasm

package pages

import (
	"github.com/vcrobe/nojs/runtime"
)

// CounterPage demonstrates reactive state via StateHasChanged().
type CounterPage struct {
	runtime.ComponentBase

	Count       int
	RenderCount int
}

func (c *CounterPage) OnParametersSet() {
	c.RenderCount++
}

func (c *CounterPage) Increment() {
	c.Count++
	c.StateHasChanged()
}

func (c *CounterPage) Decrement() {
	c.Count--
	c.StateHasChanged()
}

func (c *CounterPage) Reset() {
	c.Count = 0
	c.StateHasChanged()
}
