//go:build js || wasm
// +build js wasm

package appcomponents

import (
	"github.com/vcrobe/nojs/events"
	"github.com/vcrobe/nojs/runtime"
)

// BlogPage is the component rendered for the "/blog/{year}" route.

type BlogPage struct {
	runtime.ComponentBase

	Year int
}

// NavigateToHome handles navigation to the home page
func (a *BlogPage) NavigateToHome(e events.ClickEventArgs) {
	e.PreventDefault()
	if err := a.Navigate("/"); err != nil {
		println("Navigation error:", err.Error())
	}
}
