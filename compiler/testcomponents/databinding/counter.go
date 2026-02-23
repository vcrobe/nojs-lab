package databinding

import (
	"github.com/ForgeLogic/nojs/runtime"
)

// Counter is a simple test component that demonstrates data binding.
// This component definition works in both WASM and test environments
// thanks to the build-tag-free runtime.ComponentBase.
type Counter struct {
	runtime.ComponentBase
	Count int
	Label string
}

// Increment increases the counter and triggers a re-render.
func (c *Counter) Increment() {
	c.Count++
	c.StateHasChanged()
}

// SetLabel updates the label and triggers a re-render.
func (c *Counter) SetLabel(newLabel string) {
	c.Label = newLabel
	c.StateHasChanged()
}
