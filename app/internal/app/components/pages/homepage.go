//go:build js || wasm
// +build js wasm

package pages

import (
	"github.com/vcrobe/app/internal/app/components/shared/modal"
	"github.com/vcrobe/nojs/runtime"
)

// HomePage is the component rendered for the "/" route.
type HomePage struct {
	runtime.ComponentBase
	Years []int

	// The parent *must* control the visibility state.
	IsMyModalVisible bool
	// We can store a message to show the result
	LastModalResult string
}

func (h *HomePage) OnInit() {
	h.Years = []int{120, 300}
}

// ShowTheModal is called by our button.
func (c *HomePage) ShowTheModal() {
	c.IsMyModalVisible = true
	c.LastModalResult = "Modal is open..."
	// CRITICAL (Rule 6): We changed state, so we *must* call StateHasChanged().
	c.StateHasChanged()
}

// HandleModalClose matches the 'OnClose' prop signature.
// This is how the dialog communicates back to the parent.
func (c *HomePage) HandleModalClose(result modal.ModalResult) {
	c.IsMyModalVisible = false // Hide the dialog

	if result == modal.Ok {
		c.LastModalResult = "You clicked OK!"
	} else {
		c.LastModalResult = "You clicked Cancel."
	}

	c.StateHasChanged()
}
