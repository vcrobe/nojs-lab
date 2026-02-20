//go:build js || wasm

package pages

import (
	"github.com/vcrobe/app/internal/app/components/shared/modal"
	"github.com/vcrobe/nojs/runtime"
)

// SlotsPage demonstrates content projection via the []*vdom.VNode slot pattern.
type SlotsPage struct {
	runtime.ComponentBase

	IsModalVisible bool
	LastResult     string
	HasResult      bool
	RenderCount    int
}

func (c *SlotsPage) OnParametersSet() {
	c.RenderCount++
}

func (c *SlotsPage) OpenModal() {
	c.IsModalVisible = true
	c.LastResult = ""
	c.HasResult = false
	c.StateHasChanged()
}

func (c *SlotsPage) HandleModalClose(result modal.ModalResult) {
	c.IsModalVisible = false
	c.HasResult = true
	if result == modal.Ok {
		c.LastResult = "✅ You clicked OK"
	} else {
		c.LastResult = "❌ You clicked Cancel"
	}
	c.StateHasChanged()
}
