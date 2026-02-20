//go:build js || wasm

package pages

import (
"github.com/vcrobe/nojs/runtime"
)

// LandingPage is the home page at "/" — introduces the framework and links to all demo pages.
type LandingPage struct {
runtime.ComponentBase
}
