package runtime

import "fmt"

// ComponentBase is a struct that components can embed to gain access to the
// StateHasChanged method, which triggers a UI re-render.
// This type has no build tags and works in both WASM and test environments.
type ComponentBase struct {
	renderer Renderer // Use interface type, not concrete implementation
}

// SetRenderer is called by the framework's runtime to inject a reference
// to the renderer, enabling StateHasChanged. This method should not be
// called by user code.
func (b *ComponentBase) SetRenderer(r Renderer) {
	b.renderer = r
}

// StateHasChanged signals to the framework that the component's state has
// been updated and the UI should be re-rendered to reflect the changes.
func (b *ComponentBase) StateHasChanged() {
	if b.renderer == nil {
		println("StateHasChanged called, but renderer is nil (component not mounted?)")
		return
	}
	// Trigger a re-render of the root component.
	b.renderer.ReRender()
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
