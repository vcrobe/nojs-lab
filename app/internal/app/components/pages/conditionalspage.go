//go:build js || wasm

package pages

import (
	"github.com/vcrobe/nojs/runtime"
)

// ConditionalsPage demonstrates {@if}/{@else} conditional rendering.
type ConditionalsPage struct {
	runtime.ComponentBase

	IsLoggedIn  bool
	ShowAlert   bool
	RenderCount int
}

func (c *ConditionalsPage) OnParametersSet() {
	c.RenderCount++
}

func (c *ConditionalsPage) ToggleLogin() {
	c.IsLoggedIn = !c.IsLoggedIn
	c.StateHasChanged()
}

func (c *ConditionalsPage) ToggleAlert() {
	c.ShowAlert = !c.ShowAlert
	c.StateHasChanged()
}
