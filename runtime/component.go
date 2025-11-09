package runtime

import "github.com/vcrobe/nojs/vdom"

// Component interface defines the structure for all components in the framework.
// This interface has NO build tags, making it available to both WASM and native test builds.
// The Render method accepts the Renderer interface (not concrete type) so both WASM
// and test implementations can use it.
type Component interface {
	// Render generates the virtual DOM tree for this component.
	// The renderer parameter provides access to framework services like RenderChild.
	Render(r Renderer) *vdom.VNode

	// SetRenderer is called by the framework to attach the renderer to the component.
	// This enables StateHasChanged() to trigger re-renders.
	SetRenderer(r Renderer)
}
