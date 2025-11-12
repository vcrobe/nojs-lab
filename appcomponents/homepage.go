//go:build js || wasm
// +build js wasm

package appcomponents

import (
	"github.com/vcrobe/nojs/runtime"
)

// HomePage is the component rendered for the "/" route.
type HomePage struct {
	runtime.ComponentBase
	Years []int
}

func (h *HomePage) OnInit() {
	h.Years = []int{120, 300}
}
