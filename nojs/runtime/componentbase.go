package runtime

import (
	"fmt"

	"github.com/ForgeLogic/nojs/console"
)

// ComponentBase is a struct that components can embed to gain access to the
// StateHasChanged method, which triggers a UI re-render.
// This type has no build tags and works in both WASM and test environments.
type ComponentBase struct {
	renderer   Renderer  // Use interface type, not concrete implementation
	slotParent Component // Parent layout if this component is in a []*vdom.VNode slot
}

// SetRenderer is called by the framework's runtime to inject a reference
// to the renderer, enabling StateHasChanged. This method should not be
// called by user code.
func (b *ComponentBase) SetRenderer(r Renderer) {
	b.renderer = r
}

// GetRenderer returns the renderer instance associated with this component.
// Used internally by components that need direct access to renderer methods.
func (b *ComponentBase) GetRenderer() Renderer {
	return b.renderer
}

// StateHasChanged signals to the framework that the component's state has
// been updated and the UI should be re-rendered to reflect the changes.
// If this component is mounted inside a layout's []*vdom.VNode slot,
// triggers scoped re-render of only that slot. Otherwise, full re-render.
func (b *ComponentBase) StateHasChanged() {
	if b.renderer == nil {
		console.Error("StateHasChanged called, but renderer is nil (component not mounted?)")
		return
	}

	// Check if this component is in a layout's slot (in-memory tracking)
	if b.slotParent != nil {
		// Scoped re-render: only re-render the parent layout's slot content
		if err := b.renderer.ReRenderSlot(b.slotParent); err != nil {
			console.Error("ReRenderSlot failed:", err.Error())
		}
		return
	}

	// Full re-render (for root or unslotted components)
	b.renderer.ReRender()
}

// SetSlotParent associates this component with a parent layout.
// Called by the renderer when mounting a child into a layout's []*vdom.VNode slot.
// No DOM attributes are added—the relationship is tracked entirely in Go memory.
func (b *ComponentBase) SetSlotParent(parent Component) {
	b.slotParent = parent
}

// Navigate requests client-side navigation to a new path.
// This is used by components (such as Link) to trigger routing without full page reloads.
// The path will be passed to the router, which will update the browser URL and
// render the appropriate component.
//
// Example usage in a component:
//
//	func (c *MyComponent) HandleClick() {
//	    if err := c.Navigate("/about"); err != nil {
//	        println("Navigation failed:", err.Error())
//	    }
//	}
//
// Returns an error if the renderer is not set or navigation fails.
func (b *ComponentBase) Navigate(path string) error {
	if b.renderer == nil {
		return fmt.Errorf("navigate called, but renderer is nil (component not mounted?)")
	}
	return b.renderer.Navigate(path)
}
