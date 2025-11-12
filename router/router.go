//go:build js && wasm

package router

import (
	"strings"
	"syscall/js"

	"github.com/vcrobe/nojs/console"
	"github.com/vcrobe/nojs/runtime"
)

// RoutingMode defines how URLs are managed by the router.
type RoutingMode int

const (
	// PathMode uses HTML5 History API with clean URLs like /about
	// Requires server configuration to serve index.html for all routes.
	PathMode RoutingMode = iota

	// HashMode uses hash-based URLs like #/about
	// Works without server configuration (good for static hosting).
	HashMode
)

// RouteHandler is a function that creates a component instance for a route.
// It receives URL parameters extracted from the path (e.g., {id} in /users/{id}).
type RouteHandler func(params map[string]string) runtime.Component

// routeDefinition holds the internal representation of a registered route.
type routeDefinition struct {
	Path    string // e.g., "/users/{id}"
	Handler RouteHandler
}

// Router implements the runtime.NavigationManager interface.
// It handles client-side routing with support for both path-based and hash-based modes.
type Router struct {
	routes           []routeDefinition
	onChange         func(runtime.Component, string) // Second parameter is the path/key
	mode             RoutingMode
	notFoundHandler  RouteHandler
	popstateListener js.Func
}

// Config holds configuration options for the router.
type Config struct {
	Mode RoutingMode // PathMode or HashMode
}

// New creates a new Router with the given configuration.
// If config is nil, defaults to PathMode.
func New(config *Config) *Router {
	mode := PathMode
	if config != nil {
		mode = config.Mode
	}

	return &Router{
		routes: make([]routeDefinition, 0),
		mode:   mode,
	}
}

// Handle registers a route with its handler function.
// The path can contain parameters in curly braces, e.g., "/users/{id}".
//
// Example:
//
//	router.Handle("/", func(p map[string]string) runtime.Component {
//	    return &HomePage{}
//	})
//	router.Handle("/users/{id}", func(p map[string]string) runtime.Component {
//	    return &UserPage{UserID: p["id"]}
//	})
func (r *Router) Handle(path string, handler RouteHandler) {
	r.routes = append(r.routes, routeDefinition{
		Path:    path,
		Handler: handler,
	})
}

// HandleNotFound registers a handler for 404 (not found) cases.
// This handler is called when no route matches the current path.
//
// Example:
//
//	router.HandleNotFound(func(p map[string]string) runtime.Component {
//	    return &NotFoundPage{}
//	})
func (r *Router) HandleNotFound(handler RouteHandler) {
	r.notFoundHandler = handler
}

// Start implements runtime.NavigationManager.Start().
// It initializes the router by reading the initial URL, setting up browser
// event listeners, and calling the onChange callback with the initial component.
func (r *Router) Start(onChange func(runtime.Component, string)) error {
	r.onChange = onChange

	// Listen for browser back/forward button clicks
	r.popstateListener = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		r.handlePathChange()
		return nil
	})
	js.Global().Set("onpopstate", r.popstateListener)

	// Handle the initial page load (read current URL and render component)
	r.handlePathChange()
	return nil
}

// Navigate implements runtime.NavigationManager.Navigate().
// It changes the browser URL and renders the new component without a full page reload.
func (r *Router) Navigate(path string) error {
	if r.mode == PathMode {
		// Use HTML5 History API
		js.Global().Get("history").Call("pushState", nil, "", path)
	} else {
		// Use hash navigation
		js.Global().Get("location").Set("hash", path)
	}

	r.handlePathChange()
	return nil
}

// GetComponentForPath implements runtime.NavigationManager.GetComponentForPath().
// It matches the path against registered routes and returns the corresponding component.
func (r *Router) GetComponentForPath(path string) (runtime.Component, bool) {
	for _, route := range r.routes {
		params, matched := r.matchRoute(route.Path, path)
		if matched {
			return route.Handler(params), true
		}
	}
	return nil, false
}

// handlePathChange is called whenever the URL changes (programmatically or via browser navigation).
// It determines which component to render and calls the onChange callback.
func (r *Router) handlePathChange() {
	path := r.getCurrentPath()

	comp, found := r.GetComponentForPath(path)
	if found && r.onChange != nil {
		r.onChange(comp, path) // Pass path as key
	} else if !found && r.notFoundHandler != nil {
		// Call 404 handler
		r.onChange(r.notFoundHandler(nil), path)
	} else if r.onChange != nil {
		// No route found and no 404 handler configured
		console.Warn("[Router] No route found for path: '%s'\n", path)
	}
}

// getCurrentPath returns the current path based on the routing mode.
func (r *Router) getCurrentPath() string {
	if r.mode == PathMode {
		return js.Global().Get("location").Get("pathname").String()
	} else {
		// Hash mode: extract path after the #
		hash := js.Global().Get("location").Get("hash").String()
		if len(hash) > 1 && hash[0] == '#' {
			return hash[1:] // Remove the # prefix
		}
		return "/"
	}
}

// matchRoute checks if a URL path matches a route pattern and extracts parameters.
// Returns (params, true) if matched, or (nil, false) if not matched.
//
// Example:
//
//	matchRoute("/users/{id}", "/users/123") returns ({"id": "123"}, true)
//	matchRoute("/about", "/about") returns ({}, true)
//	matchRoute("/about", "/contact") returns (nil, false)
func (r *Router) matchRoute(routePath, urlPath string) (map[string]string, bool) {
	// Normalize paths (remove trailing slashes for comparison)
	routePath = strings.TrimSuffix(routePath, "/")
	urlPath = strings.TrimSuffix(urlPath, "/")

	// Handle root path specially
	if routePath == "" {
		routePath = "/"
	}
	if urlPath == "" {
		urlPath = "/"
	}

	routeParts := strings.Split(strings.Trim(routePath, "/"), "/")
	urlParts := strings.Split(strings.Trim(urlPath, "/"), "/")

	// Paths must have the same number of segments to match
	if len(routeParts) != len(urlParts) {
		return nil, false
	}

	params := make(map[string]string)

	for i := range routeParts {
		if strings.HasPrefix(routeParts[i], "{") && strings.HasSuffix(routeParts[i], "}") {
			// This is a parameter placeholder
			paramName := strings.Trim(routeParts[i], "{}")
			params[paramName] = urlParts[i]
		} else if routeParts[i] != urlParts[i] {
			// Static segment doesn't match
			return nil, false
		}
	}

	return params, true
}

// Cleanup releases resources held by the router.
// Call this when the router is no longer needed.
func (r *Router) Cleanup() {
	if !r.popstateListener.IsUndefined() {
		r.popstateListener.Release()
	}
}
