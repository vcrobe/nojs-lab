//go:build js || wasm

package pages

import (
	"github.com/ForgeLogic/app/internal/appstate"
	"github.com/ForgeLogic/nojs/runtime"
)

var routerDemoIDs = []string{"42", "go-wasm", "hello", "framework", "nojs", "2026"}

// RouterParamsPage demonstrates URL route parameters and programmatic navigation.
type RouterParamsPage struct {
	runtime.ComponentBase

	ID          string
	RenderCount int
}

func (c *RouterParamsPage) OnParametersSet() {
	c.RenderCount = appstate.RenderCount.Get()
	println("RouterParamsPage: OnParametersSet called with ID =", c.ID)
}

func (c *RouterParamsPage) GoToNext() {
	// update RenderCount state
	appstate.RenderCount.Set(appstate.RenderCount.Get() + 1)

	// update NextIDIndex state
	nextIdx := appstate.NextIDIndex.Get()
	next := routerDemoIDs[nextIdx%len(routerDemoIDs)]
	appstate.NextIDIndex.Set((nextIdx + 1) % len(routerDemoIDs))

	// navigate to the next ID route
	c.Navigate("/router/" + next)
}
