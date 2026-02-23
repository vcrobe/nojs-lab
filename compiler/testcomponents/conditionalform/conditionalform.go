package conditionalform

import (
	"github.com/ForgeLogic/nojs/runtime"
)

// ConditionalForm is a minimal test component that mirrors the FormsPage
// "Live Preview" pattern: when HasName is true a live-preview block is shown;
// when it is false a muted placeholder is shown instead.
//
// This component exists specifically to exercise the VNode conditional
// (if/else) rendering path and the type-then-clear regression where clearing
// the input failed to restore the muted placeholder in the DOM.
type ConditionalForm struct {
	runtime.ComponentBase

	Name    string
	HasName bool
}

// SetName simulates a user typing into the name input field.
func (c *ConditionalForm) SetName(name string) {
	c.Name = name
	c.HasName = name != ""
	c.StateHasChanged()
}
