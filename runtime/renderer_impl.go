//go:build js || wasm
// +build js wasm

package runtime

import (
	"fmt"

	"github.com/vcrobe/nojs/vdom"
)

// Compile-time assertion to ensure the concrete RendererImpl implements the Renderer interface.
var _ Renderer = (*RendererImpl)(nil)

// RendererImpl is the concrete implementation of the Renderer interface.
// It manages the component instance tree and handles rendering lifecycle.
type RendererImpl struct {
	instances        map[string]Component
	initialized      map[string]bool   // Track which components have been initialized
	activeKeys       map[string]bool   // Track which components are active in the current render
	currentComponent Component         // The currently active root component (set by router or directly)
	navManager       NavigationManager // Optional: router for client-side navigation
	mountID          string
	prevVDOM         *vdom.VNode // Previous VDOM tree for patching
}

// NewRenderer creates a new runtime renderer.
// If navManager is provided, the renderer will support client-side routing.
// If navManager is nil, the renderer works without routing (useful for non-SPA apps).
func NewRenderer(navManager NavigationManager, mountID string) *RendererImpl {
	return &RendererImpl{
		instances:   make(map[string]Component),
		initialized: make(map[string]bool),
		activeKeys:  make(map[string]bool),
		navManager:  navManager,
		mountID:     mountID,
		prevVDOM:    nil,
	}
}

// SetCurrentComponent sets the component to be rendered.
// This is typically called by the router's onChange callback when navigation occurs.
// For non-routed apps, it can be called directly with a static component.
func (r *RendererImpl) SetCurrentComponent(comp Component) {
	r.currentComponent = comp
}

// RenderRoot starts the rendering process for the entire application.
func (r *RendererImpl) RenderRoot() {
	// Reset activeKeys for this render cycle
	r.activeKeys = make(map[string]bool)

	// On each root render, we build the VDOM tree from the current component.
	// Ensure the component has a reference to the renderer for StateHasChanged and Navigate.
	if r.currentComponent != nil {
		r.currentComponent.SetRenderer(r)

		// Handle root component lifecycle
		if _, initialized := r.initialized["__root__"]; !initialized {
			// Call OnInit only once, before first render
			if initializer, ok := r.currentComponent.(Initializer); ok {
				r.callOnInit(initializer, "__root__")
			}
			r.initialized["__root__"] = true
		}

		// Call OnParametersSet before every render (including first)
		if paramReceiver, ok := r.currentComponent.(ParameterReceiver); ok {
			r.callOnParametersSet(paramReceiver, "__root__")
		}
	}

	newVDOM := r.currentComponent.Render(r)

	if r.prevVDOM == nil {
		// Initial render: clear and render fresh
		vdom.Clear(r.mountID)
		vdom.RenderToSelector(r.mountID, newVDOM)
	} else {
		// Subsequent renders: patch the existing DOM
		vdom.Patch(r.mountID, r.prevVDOM, newVDOM)
	}

	// Store the new VDOM tree for the next render cycle
	r.prevVDOM = newVDOM

	// Clean up components that were not rendered in this cycle
	r.cleanupUnmountedComponents()
}

// RenderChild is called by compiler-generated code to render a child component.
// It handles the core logic of instance creation and reuse.
func (r *RendererImpl) RenderChild(key string, childWithProps Component) *vdom.VNode {
	// Mark this component as active in the current render cycle
	r.activeKeys[key] = true

	instance, exists := r.instances[key]
	isFirstRender := false

	if !exists {
		// First time seeing this component at this location, so store the new instance.
		instance = childWithProps
		r.instances[key] = instance
		isFirstRender = true
	} else {
		// We have seen this component before. Preserve the existing instance to keep state.
		// Apply new props from childWithProps to the existing instance.
		if updater, ok := instance.(PropUpdater); ok {
			updater.ApplyProps(childWithProps)
		}
	}

	// Now, render the child (either the new or reused one).
	// Ensure the instance knows about the renderer so it can call StateHasChanged.
	instance.SetRenderer(r)

	// Call lifecycle methods in the correct order
	if isFirstRender {
		// Call OnInit only once, before first render
		if initializer, ok := instance.(Initializer); ok {
			r.callOnInit(initializer, key)
		}
		r.initialized[key] = true
	}

	// Call OnParametersSet before every render (including first)
	if paramReceiver, ok := instance.(ParameterReceiver); ok {
		r.callOnParametersSet(paramReceiver, key)
	}

	return instance.Render(r)
}

// cleanupUnmountedComponents removes components that are no longer in the tree
// and calls their OnDestroy lifecycle method if they implement the Cleaner interface.
func (r *RendererImpl) cleanupUnmountedComponents() {
	for key, instance := range r.instances {
		// If the component wasn't marked as active in this render, it's been unmounted
		if !r.activeKeys[key] {
			// Call OnDestroy if the component implements Cleaner
			if cleaner, ok := instance.(Cleaner); ok {
				r.callOnDestroy(cleaner, key)
			}

			// Remove from tracking maps
			delete(r.instances, key)
			delete(r.initialized, key)
		}
	}
}

// ReRender patches the DOM with minimal changes.
func (r *RendererImpl) ReRender() {
	r.RenderRoot()
}

// Navigate implements the Navigator interface.
// It delegates to the NavigationManager (router) to perform client-side navigation.
// Returns an error if no router is configured.
func (r *RendererImpl) Navigate(path string) error {
	if r.navManager == nil {
		return fmt.Errorf("no router configured for navigation")
	}
	return r.navManager.Navigate(path)
}
