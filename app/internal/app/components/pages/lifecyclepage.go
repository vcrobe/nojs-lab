//go:build js || wasm

package pages

import (
	"time"

	"github.com/vcrobe/nojs/runtime"
)

// lifecycleMountCount tracks total mounts across navigations (package-level).
var lifecycleMountCount int

// LifecyclePage demonstrates OnMount, OnParametersSet, and OnUnmount.
type LifecyclePage struct {
	runtime.ComponentBase

	MountTime     string
	MountCount    int
	ParamSetCount int
}

func (c *LifecyclePage) OnMount() {
	lifecycleMountCount++
	c.MountCount = lifecycleMountCount
	c.MountTime = time.Now().Format("15:04:05.000")
}

func (c *LifecyclePage) OnParametersSet() {
	c.ParamSetCount++
}

func (c *LifecyclePage) OnUnmount() {
	// Nothing to clean up here — but this hook fires when you navigate away.
	// The package-level counter persists so the next mount shows a higher number.
}
