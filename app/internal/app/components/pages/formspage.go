//go:build js || wasm

package pages

import (
	"github.com/vcrobe/nojs/events"
	"github.com/vcrobe/nojs/runtime"
)

// FormsPage demonstrates event binding and live data binding with @oninput / @onchange.
type FormsPage struct {
	runtime.ComponentBase

	Name     string
	Language string
	IsSenior bool
	HasName  bool

	RenderCount int
}

func (c *FormsPage) OnMount() {
	c.Language = "Go"
}

func (c *FormsPage) OnParametersSet() {
	c.RenderCount++
}

func (c *FormsPage) HandleNameInput(e events.ChangeEventArgs) {
	c.Name = e.Value
	c.HasName = c.Name != ""
	c.StateHasChanged()
}

func (c *FormsPage) HandleLanguageChange(e events.ChangeEventArgs) {
	c.Language = e.Value
	c.StateHasChanged()
}

func (c *FormsPage) ToggleSeniority() {
	c.IsSenior = !c.IsSenior
	c.StateHasChanged()
}
