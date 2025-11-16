package modal

import (
	"github.com/vcrobe/nojs/runtime"
	"github.com/vcrobe/nojs/vdom"
)

type Modal struct {
	runtime.ComponentBase

	// --- PROPS ---

	// IsVisible controls the dialog's display.
	// We bind this from the parent.
	IsVisible bool

	// Title text for the dialog header.
	Title string

	// Type controls which icon is shown.
	Type ModalType

	// ShowCancelButton determines if the 'Cancel' button is rendered.
	ShowCancelButton bool

	// OnClose is the function callback prop. The parent provides
	// a method that matches this signature.
	OnClose func(result ModalResult)

	// BodyContent is our "slot". The framework will
	// inject any child VNodes from the parent here.
	BodyContent []*vdom.VNode

	// --- INTERNAL STATE ---
	// (Used by the template for {@if} blocks)

	// We must use boolean fields for {@if}.
	// We'll calculate these based on the 'Type' prop.
	IsInfo    bool
	IsWarning bool
	IsError   bool
}

// OnParametersSet is called every time props are set.
// We use it to update our internal boolean fields for the template.
func (c *Modal) OnParametersSet() {
	// This logic translates the 'Type' prop into simple booleans
	// that the template's {@if} blocks can use.
	c.IsInfo = (c.Type == Information)
	c.IsWarning = (c.Type == Warning)
	c.IsError = (c.Type == Error)
}

// HandleOk is bound to the 'Ok' button's @onclick event.
func (c *Modal) HandleOk() {
	if c.OnClose != nil {
		c.OnClose(Ok)
	}
}

// HandleCancel is bound to 'Cancel' and 'X' buttons.
func (c *Modal) HandleCancel() {
	if c.OnClose != nil {
		c.OnClose(Cancel)
	}
}
