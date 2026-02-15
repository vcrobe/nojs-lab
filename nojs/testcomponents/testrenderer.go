package testcomponents

import (
	"github.com/vcrobe/nojs/runtime"
	"github.com/vcrobe/nojs/vdom"
)

// TestRenderer is a minimal test harness that implements runtime.Renderer
// for in-memory testing without browser or WASM dependencies.
//
// It captures VDOM output from component renders and allows tests to:
// - Attach components to the renderer
// - Trigger re-renders via StateHasChanged()
// - Inspect the resulting VDOM tree
type TestRenderer struct {
	currentVDOM *vdom.VNode
	component   runtime.Component
}

// Compile-time assertion to ensure TestRenderer implements runtime.Renderer interface.
var _ runtime.Renderer = (*TestRenderer)(nil)

// NewTestRenderer creates a test renderer attached to the given component.
func NewTestRenderer(comp runtime.Component) *TestRenderer {
	r := &TestRenderer{
		component: comp,
	}
	comp.SetRenderer(r)
	return r
}

// RenderRoot performs the initial render of the component.
// This should be called at the start of a test to get the initial VDOM.
func (r *TestRenderer) RenderRoot() *vdom.VNode {
	r.currentVDOM = r.component.Render(r)
	return r.currentVDOM
}

// ReRender performs a re-render of the component.
// This is called by StateHasChanged() when the component requests a re-render.
func (r *TestRenderer) ReRender() {
	r.currentVDOM = r.component.Render(r)
}

// GetCurrentVDOM returns the most recently rendered VDOM tree.
// Tests use this to inspect the component's output after renders.
func (r *TestRenderer) GetCurrentVDOM() *vdom.VNode {
	return r.currentVDOM
}

// RenderChild is a stub for child component rendering.
// For simple data binding tests, we typically won't have child components.
// If needed in the future, this can be expanded to track child instances.
func (r *TestRenderer) RenderChild(key string, child runtime.Component) *vdom.VNode {
	child.SetRenderer(r)
	return child.Render(r)
}

// Navigate is a no-op implementation for tests.
// Most tests don't need navigation functionality.
func (r *TestRenderer) Navigate(path string) error {
	// No-op for tests
	return nil
}

// ReRenderSlot patches only the BodyContent slot of a layout.
// For tests, this simply re-renders the slot parent component.
func (r *TestRenderer) ReRenderSlot(slotParent runtime.Component) error {
	// Re-render the slot parent component
	slotParent.Render(r)
	return nil
}
