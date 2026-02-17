//go:build js || wasm
// +build js wasm

package router

import (
	"fmt"
	"strings"
	"sync"
	"syscall/js"

	"github.com/vcrobe/nojs/console"
	"github.com/vcrobe/nojs/runtime"
	"github.com/vcrobe/nojs/vdom"
)

// Engine manages routing with the app shell pattern and pivot-based layout reuse.
// It preserves layout instances across navigations when the layout chain matches.
type Engine struct {
	mu               sync.Mutex
	currentPath      string
	currentRoute     *Route
	activeChain      []ComponentMetadata
	liveInstances    []runtime.Component // Parallel to activeChain; instances are reused
	pivotPoint       int                 // First index where chain differs between routes
	routes           map[string]*Route
	renderer         runtime.Renderer
	onRouteChange    func(chain []runtime.Component, key string)
	popstateListener js.Func
}

// New creates a new router engine.
// The renderer can be set later via SetRenderer if needed.
func NewEngine(renderer runtime.Renderer) *Engine {
	return &Engine{
		routes:        make(map[string]*Route),
		renderer:      renderer,
		liveInstances: make([]runtime.Component, 0, 4),
	}
}

// SetRenderer sets the renderer on the engine (used after engine creation).
func (e *Engine) SetRenderer(renderer runtime.Renderer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.renderer = renderer
}

// RegisterRoutes adds routes to the engine.
// Routes are keyed by their Path for O(1) lookup.
func (e *Engine) RegisterRoutes(routes []Route) {
	for i := range routes {
		e.routes[routes[i].Path] = &routes[i]
	}
}

// SetRouteChangeCallback sets the callback invoked when navigation occurs.
// The callback is passed the chain of component instances (from pivot onwards, including
// sublayouts and the leaf page) and a unique key for reconciliation.
func (e *Engine) SetRouteChangeCallback(fn func(chain []runtime.Component, key string)) {
	e.onRouteChange = fn
}

// Navigate changes the current route and triggers appropriate updates.
// It uses the pivot algorithm to determine which layouts can be preserved.
// If skipPushState is true, the URL won't be updated (used for popstate events).
func (e *Engine) Navigate(path string) error {
	return e.navigateInternal(path, false)
}

// navigateInternal handles the navigation logic with optional skipPushState flag.
func (e *Engine) navigateInternal(path string, skipPushState bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	console.Log("[Engine.Navigate] Called with path:", path)

	if path == "" {
		console.Warn("[Engine.Navigate] The path is empty string")
	}

	console.Log("[Engine.Navigate] Current path:", e.currentPath)

	targetRoute, ok := e.routes[path]
	if !ok {
		console.Error("[Engine.Navigate] No route found for path:", path)
		return fmt.Errorf("no route for path: %s", path)
	}

	console.Log("[Engine.Navigate] Route found")

	// Update browser history using pushState (unless this is a popstate navigation)
	if !skipPushState {
		console.Log("[Engine.Navigate] Updating URL with pushState")
		history := js.Global().Get("history")
		history.Call("pushState", nil, "", path)
		console.Log("[Engine.Navigate] URL updated, current location:", js.Global().Get("location").Get("pathname").String())
	} else {
		console.Log("[Engine.Navigate] Skipping pushState (popstate event)")
	}

	// Calculate pivot point: first index where TypeID differs
	pivot := e.calculatePivot(targetRoute.Chain)

	console.Log("[Engine.Navigate] Pivot point:", pivot, "Chain length:", len(targetRoute.Chain))

	// Extract URL parameters from route pattern
	params := e.extractParams(targetRoute.Path, path)
	console.Log("[Engine.Navigate] Extracted params:", fmt.Sprintf("%v", params))

	// Destroy volatile (new) component instances from pivot onwards
	for i := pivot; i < len(e.liveInstances); i++ {
		instance := e.liveInstances[i]

		// Clear slot parent reference to break circular references
		if slotTracking, ok := interface{}(instance).(interface{ SetSlotParent(runtime.Component) }); ok {
			slotTracking.SetSlotParent(nil)
		}
	}

	// Instantiate new chain segment (from pivot onwards)
	newInstances := make([]runtime.Component, len(targetRoute.Chain))

	// Copy stable instances (before pivot)
	copy(newInstances[:pivot], e.liveInstances[:pivot])

	// Create new instances from pivot onwards
	for i := pivot; i < len(targetRoute.Chain); i++ {
		instance := targetRoute.Chain[i].Factory(params)

		// Inject renderer so component can call StateHasChanged() and Navigate()
		instance.SetRenderer(e.renderer)

		newInstances[i] = instance
	}

	// Link chain: inject each child into parent's BodyContent slot
	// Skip this if using AppShell pattern (onRouteChange callback set) to prevent double-rendering
	if e.onRouteChange == nil {
		for i := 0; i < len(newInstances)-1; i++ {
			parent := newInstances[i]
			child := newInstances[i+1]

			// Render child to VDOM and inject into parent's slot
			childVNode := child.Render(e.renderer)
			if childVNode != nil {
				// Use duck typing to set slot content - any layout with SetBodyContent method
				if layout, ok := parent.(interface{ SetBodyContent([]*vdom.VNode) }); ok {
					layout.SetBodyContent([]*vdom.VNode{childVNode})
				}
			}

			// Mark child as being in parent's slot (for scoped re-renders)
			if slotTracking, ok := interface{}(child).(interface{ SetSlotParent(runtime.Component) }); ok {
				slotTracking.SetSlotParent(parent)
			}
		}
	}

	// Notify route change callback to update AppShell state.
	// AppShell.SetPage will call StateHasChanged() to trigger a re-render.
	// The RenderChild mechanism ensures layouts are reused efficiently,
	// and VDOM patching only updates what actually changed.
	if e.onRouteChange != nil {
		// Pass the full chain to AppShell so it can handle all layers correctly
		// This includes the root layout, any preserved sublayouts, and new components
		key := fmt.Sprintf("%s:%d", path, pivot) // Unique key includes pivot for reconciliation
		console.Log("[Engine.Navigate] Calling onRouteChange with", len(newInstances), "components, key:", key)
		e.onRouteChange(newInstances, key)
		console.Log("[Engine.Navigate] AppShell will handle rendering via StateHasChanged")

		// Update state and return - AppShell's StateHasChanged handles the rendering
		e.currentPath = path
		e.currentRoute = targetRoute
		e.activeChain = targetRoute.Chain
		e.liveInstances = newInstances
		e.pivotPoint = pivot
		return nil
	}
	// Fallback: if no callback (non-AppShell apps), do scoped update
	// Trigger update at pivot boundary
	if pivot > 0 {
		// Scoped update: parent layout re-renders with new slot content
		// Only the slot subtree is diffed/patched (efficient!)
		e.renderer.ReRenderSlot(newInstances[pivot-1])
	} else {
		// Full re-render: new root layout
		e.renderer.ReRender()
	}

	// Update state
	e.currentPath = path
	e.currentRoute = targetRoute
	e.activeChain = targetRoute.Chain
	e.liveInstances = newInstances
	e.pivotPoint = pivot

	return nil
}

// calculatePivot finds the first index where current and target chains differ by TypeID.
// All components before the pivot point have matching TypeIDs and are preserved.
// All components at or after the pivot point are recreated.
func (e *Engine) calculatePivot(targetChain []ComponentMetadata) int {
	minLen := len(e.activeChain)
	if len(targetChain) < minLen {
		minLen = len(targetChain)
	}

	// Compare TypeIDs from root to leaf
	for i := 0; i < minLen; i++ {
		if e.activeChain[i].TypeID != targetChain[i].TypeID {
			return i // First mismatch is pivot point
		}
	}

	// All matched up to shorter chain length
	return minLen
}

// extractParams parses URL parameters from a path based on route pattern.
// Returns a map of parameter names to their values extracted from the URL.
//
// Example:
//
//	extractParams("/blog/{year}", "/blog/2026") returns {"year": "2026"}
//	extractParams("/users/{id}/posts/{postId}", "/users/42/posts/100") returns {"id": "42", "postId": "100"}
func (e *Engine) extractParams(routePath, actualPath string) map[string]string {
	// Normalize paths (remove trailing slashes for comparison)
	routePath = strings.TrimSuffix(routePath, "/")
	actualPath = strings.TrimSuffix(actualPath, "/")

	// Handle root path specially
	if routePath == "" {
		routePath = "/"
	}
	if actualPath == "" {
		actualPath = "/"
	}

	routeParts := strings.Split(strings.Trim(routePath, "/"), "/")
	actualParts := strings.Split(strings.Trim(actualPath, "/"), "/")

	params := make(map[string]string)

	// Extract parameters from matching segments
	for i := range routeParts {
		if i >= len(actualParts) {
			break
		}
		if strings.HasPrefix(routeParts[i], "{") && strings.HasSuffix(routeParts[i], "}") {
			// This is a parameter placeholder - extract the parameter name and value
			paramName := strings.Trim(routeParts[i], "{}")
			params[paramName] = actualParts[i]
		}
	}

	return params
}

// CurrentPath returns the current route path.
func (e *Engine) CurrentPath() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.currentPath
}

// CurrentPivotPoint returns the pivot point from the last navigation.
func (e *Engine) CurrentPivotPoint() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.pivotPoint
}

// Start initializes the router and handles browser history.
// The onChange callback is invoked when navigation occurs to update the renderer's
// current component. This implements the NavigationManager interface.
func (e *Engine) Start(onChange func(chain []runtime.Component, key string)) error {
	e.mu.Lock()
	e.onRouteChange = onChange
	e.mu.Unlock()

	// Set up popstate listener for browser back/forward buttons
	e.popstateListener = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		console.Log("[Engine] popstate event fired")
		// Read current path from browser
		currentPath := js.Global().Get("location").Get("pathname").String()
		console.Log("[Engine] popstate path:", currentPath)
		// Navigate without pushing state (URL already changed)
		e.navigateInternal(currentPath, true)
		return nil
	})
	js.Global().Call("addEventListener", "popstate", e.popstateListener)
	console.Log("[Engine] popstate listener registered")

	// Navigate to the current browser path on initial load
	initialPath := js.Global().Get("location").Get("pathname").String()
	console.Log("[Engine.Start] Initial path:", initialPath)
	if initialPath == "" {
		initialPath = "/"
	}
	return e.Navigate(initialPath)
}

// GetComponentForPath resolves a URL path to its component.
// This implements the NavigationManager interface.
// Returns the leaf component (page) for the route, or nil if not found.
func (e *Engine) GetComponentForPath(path string) (runtime.Component, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	targetRoute, ok := e.routes[path]
	if !ok {
		return nil, false
	}

	if len(targetRoute.Chain) == 0 {
		return nil, false
	}

	// Extract URL parameters from route pattern
	params := e.extractParams(targetRoute.Path, path)

	// Return the leaf component (last in chain)
	// Create a new instance to return
	leaf := targetRoute.Chain[len(targetRoute.Chain)-1]
	return leaf.Factory(params), true
}

// Cleanup releases resources held by the engine.
// Call this when the engine is no longer needed to prevent memory leaks.
func (e *Engine) Cleanup() {
	if !e.popstateListener.IsUndefined() {
		js.Global().Call("removeEventListener", "popstate", e.popstateListener)
		e.popstateListener.Release()
		console.Log("[Engine] popstate listener cleaned up")
	}
}
