//go:build js || wasm

package pages

import (
	"github.com/vcrobe/nojs/runtime"
)

var routerDemoIDs = []string{"42", "go-wasm", "hello", "framework", "nojs", "2026"}

// RouterParamsPage demonstrates URL route parameters and programmatic navigation.
type RouterParamsPage struct {
	runtime.ComponentBase

	ID          string
	RenderCount int

	nextIDIndex int
}

func (c *RouterParamsPage) OnParametersSet() {
	c.RenderCount++
}

func (c *RouterParamsPage) GoToNext() {
	next := routerDemoIDs[c.nextIDIndex%len(routerDemoIDs)]
	c.nextIDIndex++
	c.Navigate("/demo/router/" + next)
}
