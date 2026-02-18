# Router Architecture and Implementation

## Overview

The No-JS framework implements a sophisticated **Router Engine** for client-side routing with layout chain management and pivot-based component reuse. The router integrates seamlessly with the Virtual DOM (VDOM) and component lifecycle system to enable Single Page Application (SPA) navigation without full page reloads.

The Router Engine uses **HTML5 History API** with clean URLs like `/about` and `/users/123`, requiring server configuration to serve `index.html` for all routes.

This document covers the router's integration patterns, event handling, and VDOM patching challenges. For detailed information about layout hierarchies, the pivot algorithm, and the AppShell pattern, see [ROUTER_ENGINE_LAYOUTS.md](ROUTER_ENGINE_LAYOUTS.md).

---

## Table of Contents

1. [Architecture and Design Principles](#architecture-and-design-principles)
2. [Core Interfaces](#core-interfaces)
3. [Route Matching and Parameter Extraction](#route-matching-and-parameter-extraction)
4. [Integration with Renderer](#integration-with-renderer)
5. [Component Navigation](#component-navigation)
6. [Event System Integration](#event-system-integration)
7. [VDOM Event Listener Management](#vdom-event-listener-management)
8. [Browser History Integration](#browser-history-integration)
9. [Lifecycle and Initialization](#lifecycle-and-initialization)
10. [Usage Examples](#usage-examples)
11. [Technical Challenges and Solutions](#technical-challenges-and-solutions)

---

## Architecture and Design Principles

### Design Philosophy

The router architecture follows three key principles:

1. **Router Agnostic**: The framework core (`runtime` package) doesn't depend on any specific router implementation
2. **Pluggable**: Any router that implements the `NavigationManager` interface can be used
3. **Layout-Aware**: The Engine preserves layout component instances across navigations using the pivot algorithm

### Separation of Concerns

```
┌─────────────────┐
│   Application   │
│    (main.go)    │
└────────┬────────┘
         │
         ├──────────────────┐
         │                  │
         ▼                  ▼
┌────────────────┐   ┌──────────────┐
│ Router Engine  │   │   Renderer   │
│   (router/)    │◄──┤  (runtime/)  │
└────────────────┘   └──────┬───────┘
         │                   │
         │                   ▼
         │            ┌──────────────┐
         │            │  Components  │
         │            │ (ComponentBase)│
         │            └──────────────┘
         │
         ▼
┌────────────────┐
│  Browser APIs  │
│ (History API)  │
└────────────────┘
```

The router is **injected** into the renderer at initialization, allowing the framework to work with or without routing.

---

## Core Interfaces

### NavigationManager Interface

Defined in `runtime/navigation.go`, this is the contract that any router must implement:

```go
type NavigationManager interface {
    // Start initializes the router with an onChange callback
    Start(onChange func(chain []Component, key string)) error
    
    // Navigate programmatically changes the URL and renders new component
    Navigate(path string) error
    
    // GetComponentForPath resolves a path to its component
    GetComponentForPath(path string) (Component, bool)
}
```

**Key Responsibilities**:
- Initialize browser event listeners (popstate for back/forward buttons)
- Read initial URL on application startup
- Match URL paths to registered routes
- Create component instances via route handlers
- Call the `onChange` callback to trigger rendering with a chain of components and a unique key

### Navigator Interface

Defined in `runtime/navigation.go`, this is provided to components:

```go
type Navigator interface {
    Navigate(path string) error
}
```

**Implementation Chain**:
```
Component.Navigate() → ComponentBase.Navigate() → Renderer.Navigate() → Router.Navigate()
```

This chain allows components to trigger navigation without direct coupling to the router.

---

## Route Matching and Parameter Extraction

### Pattern Matching

The Router Engine's `matchesPattern()` and `extractParams()` functions handle pattern matching with parameter extraction:

```go
func (e *Engine) matchesPattern(pattern, path string) bool
func (e *Engine) extractParams(routePath, actualPath string) map[string]string
```

**Algorithm**:

1. **Normalize paths**: Remove trailing slashes, handle empty strings as `/`
2. **Split into segments**: Split on `/` delimiter
3. **Length check**: Routes must have same number of segments
4. **Segment-by-segment comparison**:
   - Static segments must match exactly
   - Dynamic segments (wrapped in `{}`) capture the URL value
5. **Return** extracted parameters

**Examples**:

```go
// Static route
matchesPattern("/about", "/about") → true
extractParams("/about", "/about") → map[]{}

// Dynamic route
matchesPattern("/users/{id}", "/users/123") → true
extractParams("/users/{id}", "/users/123") → map["id": "123"]

// Multi-parameter route
matchesPattern("/posts/{year}/{month}/{slug}", "/posts/2024/11/hello") → true
extractParams("/posts/{year}/{month}/{slug}", "/posts/2024/11/hello") 
  → map["year": "2024", "month": "11", "slug": "hello"]

// No match
matchesPattern("/about", "/contact") → false
```

### URL Parameter Methods

#### 1. Path Parameters (Currently Implemented) ✅

Parameters embedded directly in the URL path using `{paramName}` syntax.

**Supported:**
- Dynamic segments with curly braces (e.g., `{id}`, `{year}`, `{slug}`)
- Multiple parameters per route (e.g., `/posts/{year}/{month}/{slug}`)
- Parameters extracted via `extractParams()` and passed to `ComponentFactory` as `map[string]string`

**Examples:**
```go
// Route definition
Path: "/blog/{year}"
Path: "/users/{id}"
Path: "/posts/{year}/{month}/{slug}"

// Extracted parameters
"/blog/2026"           → {"year": "2026"}
"/users/123"           → {"id": "123"}
"/posts/2024/11/hello" → {"year": "2024", "month": "11", "slug": "hello"}
```

#### 2. Query Parameters (Not Implemented) ❌

Parameters appended after `?` in the URL for optional filters and pagination.

**Planned syntax:**
```go
"/search?q=golang&page=2"           → {"q": "golang", "page": "2"}
"/products?category=books&sort=asc" → {"category": "books", "sort": "asc"}
"/users/123?tab=profile&edit=true"  → path: {"id": "123"}, query: {"tab": "profile", "edit": "true"}
```

**Implementation approach:** Use JavaScript `URLSearchParams` API to extract query string parameters.

#### 3. Hash Fragment Parameters (Not Implemented) ❌

Parameters after `#` for in-page navigation and SPA state.

**Planned syntax:**
```go
"/users/123#comments"  → path: "/users/123", hash: "comments"
"/page#section=profile" → path: "/page", hash: "section=profile"
```

#### 4. Optional Parameters (Not Implemented) ❌

Path segments that may or may not be present.

**Planned syntax:**
```go
Path: "/blog/{year?}/{month?}"

// All match same route:
"/blog"           → {}
"/blog/2026"      → {"year": "2026"}
"/blog/2026/11"   → {"year": "2026", "month": "11"}
```

#### 5. Wildcard/Catch-All Parameters (Not Implemented) ❌

Capture remaining path segments as a single parameter.

**Planned syntax:**
```go
Path: "/files/{*filepath}"

"/files/docs/manual.pdf"        → {"filepath": "docs/manual.pdf"}
"/files/images/2024/photo.jpg"  → {"filepath": "images/2024/photo.jpg"}
```

#### 6. Parameter Constraints (Not Implemented) ❌

Type validation or regex patterns for parameters.

**Planned syntax:**
```go
Path: "/users/{id:int}"                    // id must be numeric
Path: "/posts/{slug:regex([a-z-]+)}"       // slug matches pattern
Path: "/blog/{year:range(2000,2030)}"      // year in range

"/users/123"  ✅ Valid
"/users/abc"  ❌ Constraint fails
```

#### 7. Matrix Parameters (Not Implemented) ❌

Parameters within path segments using `;` delimiter (uncommon but valid).

**Planned syntax:**
```go
"/products;color=red;size=large"     → {"color": "red", "size": "large"}
"/users;id=123;role=admin/profile"   → {"id": "123", "role": "admin"}
```

#### 8. Route State via history.pushState (Not Implemented) ❌

Hidden parameters passed via browser history API without showing in URL.

**Planned syntax:**
```go
history.Call("pushState", 
    map[string]interface{}{
        "userId": 123,
        "modal": "open",
    }, 
    "", 
    path)
```

**Use case:** Passing temporary UI state (modals, scroll position) without URL pollution.

---

### Implementation Priority

| Method | Priority | Status | Use Case |
|--------|----------|--------|----------|
| Path Parameters | - | ✅ Implemented | RESTful resource identifiers |
| Query Parameters | High | ❌ Planned | Optional filters, pagination, search |
| Hash Fragments | Medium | ❌ Planned | In-page navigation, SPA state |
| Optional Parameters | Medium | ❌ Planned | Flexible route matching |
| Wildcard Parameters | Low | ❌ Planned | File paths, nested routes |
| Parameter Constraints | Low | ❌ Planned | Type safety, validation |
| Matrix Parameters | Very Low | ❌ Planned | Complex filtering (rare) |
| Route State | Low | ❌ Planned | Hidden UI state |

---

## Integration with Renderer

### Renderer Initialization

The renderer accepts a `NavigationManager` (the Router Engine):

```go
routerEngine := router.NewEngine(nil)
renderer := runtime.NewRenderer(routerEngine, "#app")
routerEngine.SetRenderer(renderer)
```

If `nil` is passed as the navigation manager, the renderer works without routing (useful for non-SPA apps or embedded components).

### onChange Callback

The application defines how to respond to navigation using the Engine's callback:

```go
routerEngine.Start(func(chain []runtime.Component, key string) {
    appShell.SetPage(chain, key)
})
```

With the Engine, the callback receives:
- `chain`: Array of component instances (from pivot onwards, including preserved layouts and new components)
- `key`: Unique identifier for the navigation (typically `path:pivotIndex`)

**Execution Flow**:

```
URL Change → Engine.navigateInternal() 
          → calculatePivot() 
          → Instantiate new components from pivot
          → onChange(chain, key) 
          → AppShell.SetPage() 
          → AppShell.StateHasChanged()
          → Renderer.ReRender()
          → VDOM Patching
```

### SetCurrentComponent

Located in `runtime/renderer_impl.go`:

```go
func (r *RendererImpl) SetCurrentComponent(comp Component, key string) {
    r.currentComponent = comp
    r.currentKey = key
}
```

This swaps out the root component **without destroying the renderer instance**, preserving:
- Component instance cache (`r.instances`)
- Previous VDOM tree (`r.prevVDOM`)
- Lifecycle tracking (`r.initialized`, `r.activeKeys`)

The `key` parameter helps the renderer track component identity for efficient reconciliation.

---

## Component Navigation

### ComponentBase.Navigate()

Every component that embeds `runtime.ComponentBase` can trigger navigation:

```go
type MyComponent struct {
    runtime.ComponentBase
}

func (c *MyComponent) HandleClick() {
    c.Navigate("/about")
}
```

**Implementation** (`runtime/componentbase.go`):

```go
func (b *ComponentBase) Navigate(path string) error {
    if b.renderer == nil {
        return fmt.Errorf("renderer is nil (component not mounted?)")
    }
    return b.renderer.Navigate(path)
}
```

**Flow**:

1. Component calls `Navigate(path)`
2. ComponentBase delegates to `renderer.Navigate(path)`
3. Renderer delegates to `navManager.Navigate(path)`
4. Router updates browser URL and calls `onChange` callback
5. Renderer re-renders with new component

**Error Handling**: Returns error if renderer not set (component not mounted yet).

---

## Event System Integration

### EventBase Composition Pattern

All event argument types embed `EventBase` to provide common functionality:

```go
type EventBase struct {
    jsEvent               js.Value
    preventDefaultCalled  bool
    stopPropagationCalled bool
}

func (e *EventBase) PreventDefault() {
    if !e.preventDefaultCalled {
        e.jsEvent.Call("preventDefault")
        e.preventDefaultCalled = true
    }
}
```

### ClickEventArgs

Used by Link component and other click handlers:

```go
type ClickEventArgs struct {
    EventBase  // Embedded for PreventDefault/StopPropagation
    Button     int
    ClientX    int
    ClientY    int
    AltKey     bool
    CtrlKey    bool
    ShiftKey   bool
}
```

### Dual Signature Support

The `onclick` event supports two handler signatures:

```go
// No arguments (for simple actions)
func (c *Component) HandleClick() { ... }

// With event args (for advanced handling)
func (c *Component) HandleClick(e events.ClickEventArgs) { ... }
```

This is validated at **compile time** by the AOT compiler (`compiler/compiler.go`).

### Event Adapters

Located in `events/adapters.go`, these convert Go functions to JavaScript callbacks:

```go
func AdaptClickEvent(handler func(events.ClickEventArgs)) func(js.Value) {
    return func(jsEvent js.Value) {
        eventBase := NewEventBase(jsEvent)
        args := ClickEventArgs{
            EventBase: eventBase,
            Button:    jsEvent.Get("button").Int(),
            ClientX:   jsEvent.Get("clientX").Int(),
            ClientY:   jsEvent.Get("clientY").Int(),
            // ... extract other properties
        }
        handler(args)
    }
}
```

**Flow**:

```
DOM Event → js.FuncOf wrapper → AdaptClickEvent 
         → Extract event properties 
         → Call Go handler with ClickEventArgs
```

---

## VDOM Event Listener Management

### The Challenge

One of the most complex aspects of the router implementation was handling event listeners during VDOM patching. The problem:

**Naive approach**: Just call `addEventListener` during patching
**Result**: Event listeners accumulate on every navigation, causing handlers to fire multiple times

### Root Cause

JavaScript's `addEventListener()` **does not remove old listeners automatically**. When the same element is patched multiple times:

```javascript
element.addEventListener('click', handler1);  // Navigation 1
element.addEventListener('click', handler2);  // Navigation 2
// Now clicking fires BOTH handler1 and handler2!
```

### Solutions Considered

#### ❌ Solution 1: Track and Remove Individual Listeners

```go
// Store references to js.Func callbacks
// Call removeEventListener for each old listener
// Add new listeners
```

**Problems**:
- Must maintain a separate map of element → listeners
- `js.Func` references must be stored to call `.Release()`
- Complex bookkeeping, error-prone

#### ❌ Solution 2: Compare Old and New Handlers

```go
if oldVNode.Attributes["onclick"] != newVNode.Attributes["onclick"] {
    // Only re-attach if changed
}
```

**Problems**:
- **Functions cannot be compared in Go** (`panic: comparing uncomparable type func(js.Value)`)
- Even with workarounds, determining "sameness" is impossible (closures have different addresses)

#### ✅ Solution 3: Clone Element to Remove All Listeners

**Implementation** (`vdom/render.go`):

```go
func patchElement(domElement js.Value, oldVNode, newVNode *VNode) {
    // ... update attributes first ...
    
    // Check if new VNode has event handlers
    hasEventHandlers := false
    if newVNode.Attributes != nil {
        for key := range newVNode.Attributes {
            if len(key) > 2 && key[0] == 'o' && key[1] == 'n' {
                hasEventHandlers = true
                break
            }
        }
    }
    
    // Clone element to remove all listeners
    if hasEventHandlers {
        cloned := domElement.Call("cloneNode", false)  // false = don't clone children
        
        // Move children to cloned element
        for domElement.Get("firstChild").Truthy() {
            cloned.Call("appendChild", domElement.Get("firstChild"))
        }
        
        // Replace in DOM
        parent := domElement.Get("parentNode")
        if parent.Truthy() {
            parent.Call("replaceChild", cloned, domElement)
        }
        
        // Attach fresh listeners to cloned element
        attachEventListeners(cloned, newVNode.Attributes)
        
        return  // Skip remaining patching since children already moved
    }
    
    // ... continue with normal patching ...
}
```

**Why This Works**:

1. `cloneNode(false)` creates a **shallow clone** without children or event listeners
2. We manually move children from original to clone using `appendChild`
3. `replaceChild()` swaps the elements in the DOM
4. We attach **fresh listeners** to the clean clone
5. The original element (with accumulated listeners) is garbage collected

**Performance Note**: Cloning is surprisingly efficient in modern browsers. The overhead is minimal compared to the cost of event handler bugs.

### attachEventListeners Implementation

Located in `vdom/render.go`:

```go
func attachEventListeners(domElement js.Value, attributes map[string]any) {
    if attributes == nil {
        return
    }

    for key, value := range attributes {
        if len(key) > 2 && key[0] == 'o' && key[1] == 'n' {
            eventType := key[2:]  // "onclick" → "click"
            
            // Convert Go handler to JavaScript callback
            handler, ok := value.(func(js.Value))
            if !ok {
                continue
            }
            
            cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
                if len(args) > 0 {
                    handler(args[0])
                }
                return nil
            })
            
            domElement.Call("addEventListener", eventType, cb)
            
            // TODO: Store cb somewhere to release later if needed
        }
    }
}
```

**Note**: The `js.FuncOf` callbacks are currently **not explicitly released**. This is acceptable because:
- They live as long as the DOM element exists
- When the element is removed from DOM, it becomes unreachable
- Go's garbage collector will eventually clean them up
- For long-running SPAs, a future enhancement could track and release them

---

## Browser History Integration

### Popstate Event Listener

The Router Engine listens for browser back/forward button clicks:

```go
func (e *Engine) Start(onChange func(chain []runtime.Component, key string)) error {
    e.onRouteChange = onChange
    
    // Set up popstate listener for browser back/forward buttons
    e.popstateListener = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        console.Log("[Engine] popstate event fired")
        // Read current path from browser
        currentPath := js.Global().Get("location").Get("pathname").String()
        // Navigate without pushing state (URL already changed)
        e.navigateInternal(currentPath, true)
        return nil
    })
    js.Global().Call("addEventListener", "popstate", e.popstateListener)
    
    // Navigate to the current browser path on initial load
    initialPath := js.Global().Get("location").Get("pathname").String()
    if initialPath == "" {
        initialPath = "/"
    }
    return e.Navigate(initialPath)
}
```

**Flow**:

```
User clicks back button → Browser fires 'popstate' event 
                       → Engine.navigateInternal(path, skipPushState=true)
                       → calculatePivot()
                       → Instantiate components from pivot
                       → onChange(chain, key) 
                       → AppShell updates and re-renders
```

### Cleanup

The Engine provides cleanup to release the listener:

```go
func (e *Engine) Cleanup() {
    if !e.popstateListener.IsUndefined() {
        js.Global().Call("removeEventListener", "popstate", e.popstateListener)
        e.popstateListener.Release()
    }
}
```

**Important**: In typical WASM applications that run for the entire page lifetime, cleanup is rarely needed. However, it's essential for:
- Testing scenarios
- Hot-reloading during development
- Embedding WASM modules that can be unloaded

---

## Lifecycle and Initialization

### Application Startup Sequence

1. **main.go**: Create context and persistent layout instances
2. **main.go**: Create Router Engine with `router.NewEngine(nil)`
3. **main.go**: Create renderer with `runtime.NewRenderer(routerEngine, "#app")`
4. **main.go**: Set renderer on engine with `routerEngine.SetRenderer(renderer)`
5. **main.go**: Register routes via `routerEngine.RegisterRoutes([]router.Route{...})`
6. **main.go**: Create AppShell wrapping the main layout
7. **main.go**: Set AppShell as current component
8. **main.go**: Call `routerEngine.Start(func(chain, key) { appShell.SetPage(chain, key) })`
9. **Engine**: Read initial browser URL
10. **Engine**: Call `Navigate(initialPath)`
11. **Engine**: Calculate pivot (0 for initial load)
12. **Engine**: Instantiate component chain
13. **Engine**: Call `onChange(chain, key)`
14. **AppShell**: Call `SetPage()` and `StateHasChanged()`
15. **Renderer**: Call `ReRender()`
16. **Renderer**: Inject renderer reference into components via `SetRenderer()`
17. **Renderer**: Call component lifecycle methods (`OnMount`)
18. **Renderer**: Call `component.Render()` for each component
19. **VDOM**: Render initial DOM
20. **Application**: Enter event loop (`select {}`)

### Navigation Sequence

1. **User action**: Call `component.Navigate()` from an event handler
2. **ComponentBase.Navigate()**: Delegate to `renderer.Navigate()`
3. **Renderer.Navigate()**: Delegate to `engine.Navigate()`
4. **Engine.Navigate()**: Call `history.pushState()`
5. **Engine.navigateInternal()**: Match route and calculate pivot
6. **Engine**: Destroy components at or after pivot (call `OnUnmount()`)
7. **Engine**: Copy preserved instances before pivot
8. **Engine**: Instantiate new components from pivot onwards
9. **Engine**: Inject renderer and call `OnMount()` on new components
10. **Engine**: Call `onChange(chain, key)` with component chain
11. **AppShell**: Call `SetPage()` and `StateHasChanged()`
12. **Renderer**: Call `ReRender()` (scoped to AppShell)  
13. **Renderer**: Call component lifecycle methods
14. **VDOM**: Patch DOM with minimal changes
15. **VDOM**: Clone elements with event handlers
16. **VDOM**: Attach fresh event listeners

---

## Usage Examples

### Basic Setup with Engine and AppShell

```go
func main() {
    // Create shared context
    mainLayoutCtx := &context.MainLayoutCtx{
        Title: "My App",
    }
    
    // Create persistent main layout instance (app shell)
    mainLayout := &sharedlayouts.MainLayout{
        MainLayoutCtx: mainLayoutCtx,
    }
    
    // Create the router engine
    routerEngine := router.NewEngine(nil)
    
    // Create the renderer with the engine
    renderer := runtime.NewRenderer(routerEngine, "#app")
    routerEngine.SetRenderer(renderer)
    
    // Register routes with layout chains
    routerEngine.RegisterRoutes([]router.Route{
        {
            Path: "/",
            Chain: []router.ComponentMetadata{
                {
                    Factory: func(params map[string]string) runtime.Component { return mainLayout },
                    TypeID:  MainLayout_TypeID,
                },
                {
                    Factory: func(params map[string]string) runtime.Component { return &HomePage{} },
                    TypeID:  HomePage_TypeID,
                },
            },
        },
        {
            Path: "/about",
            Chain: []router.ComponentMetadata{
                {
                    Factory: func(params map[string]string) runtime.Component { return mainLayout },
                    TypeID:  MainLayout_TypeID,
                },
                {
                    Factory: func(params map[string]string) runtime.Component { return &AboutPage{} },
                    TypeID:  AboutPage_TypeID,
                },
            },
        },
    })
    
    // Create AppShell to wrap the router's page rendering
    appShell := core.NewAppShell(mainLayout)
    renderer.SetCurrentComponent(appShell, "app-shell")
    renderer.ReRender()
    
    // Start the router with AppShell callback
    routerEngine.Start(func(chain []runtime.Component, key string) {
        appShell.SetPage(chain, key)
    })
    
    select {}
}
```

### Routes with Parameters

```go
routerEngine.RegisterRoutes([]router.Route{
    {
        Path: "/users/{id}",
        Chain: []router.ComponentMetadata{
            {
                Factory: func(params map[string]string) runtime.Component { return mainLayout },
                TypeID:  MainLayout_TypeID,
            },
            {
                Factory: func(params map[string]string) runtime.Component {
                    return &UserProfilePage{UserID: params["id"]}
                },
                TypeID: UserProfilePage_TypeID,
            },
        },
    },
    {
        Path: "/blog/{year}",
        Chain: []router.ComponentMetadata{
            {
                Factory: func(params map[string]string) runtime.Component { return mainLayout },
                TypeID:  MainLayout_TypeID,
            },
            {
                Factory: func(params map[string]string) runtime.Component {
                    year := 2026 // Default value
                    if yearStr, ok := params["year"]; ok {
                        if parsed, err := strconv.Atoi(yearStr); err == nil {
                            year = parsed
                        }
                    }
                    return &BlogPage{Year: year}
                },
                TypeID: BlogPage_TypeID,
            },
        },
    },
})
```

### Component with Navigation

```go
type AboutPage struct {
    runtime.ComponentBase
}

func (a *AboutPage) NavigateToHome(e events.ClickEventArgs) {
    e.PreventDefault()
    a.Navigate("/")
}

func (a *AboutPage) Render(r *runtime.Renderer) *vdom.VNode {
    return vdom.Div(nil,
        vdom.H1(nil, "About Page"),
        vdom.A(map[string]any{
            "href": "/",
            "onclick": events.AdaptClickEvent(a.NavigateToHome),
        }, "Back to Home"),
    )
}
```

---

## Technical Challenges and Solutions

### Challenge 1: Function Comparison in Go

**Problem**: Go doesn't allow comparing functions with `==` or `!=`

**Solution**: Don't compare handlers at all. Always re-attach listeners when they exist by cloning the element.

### Challenge 2: Event Listener Accumulation

**Problem**: `addEventListener` doesn't remove old listeners

**Solution**: Clone element to strip all listeners before attaching new ones.

### Challenge 3: Preserving Component State Across Navigation

### Challenge 3: Preserving Component State Across Navigation

**Problem**: Creating new component instances on every navigation loses state

**Solution**: The Router Engine uses the **pivot algorithm** to preserve layout instances. Only components at or after the pivot point are destroyed and recreated; layouts before the pivot are reused, maintaining their complete state.

### Challenge 4: Server Configuration Requirements

**Problem**: Direct URL access (e.g., `example.com/about`) returns 404 without server config

**Solution**: 
- Document server requirements clearly (serve `index.html` for all routes)
- Example server configs in documentation (Nginx, Apache, Go http.FileServer)
- Error messages guide developers to configure their servers properly

### Challenge 5: Preventing Memory Leaks from js.Func

**Problem**: Every `js.FuncOf` creates a callback that must be released

**Solution**: 
- **Current**: Cloning elements naturally garbage-collects old listeners
- **Future**: Implement explicit tracking and release mechanism
- **Cleanup**: Provide `Engine.Cleanup()` for popstate listener

---

## Future Enhancements

### Phase 1: Query Parameter Support (High Priority)

Add support for URL query strings to enable filtering, pagination, and search.

**Implementation:**
```go
func (e *Engine) extractQueryParams(url string) map[string]string {
    jsURL := js.Global().Get("URL").New(url, js.Global().Get("location").Get("href"))
    searchParams := jsURL.Get("searchParams")
    
    params := make(map[string]string)
    iterator := searchParams.Call("entries")
    for {
        next := iterator.Call("next")
        if next.Get("done").Bool() {
            break
        }
        entry := next.Get("value")
        params[entry.Index(0).String()] = entry.Index(1).String()
    }
    return params
}
```

**Usage:**
```go
routerEngine.RegisterRoutes([]router.Route{
    {
        Path: "/search",
        Chain: []router.ComponentMetadata{
            {
                Factory: func(params map[string]string) runtime.Component {
                    // params now includes both path and query parameters
                    query := params["q"]
                    page := params["page"]
                    return &SearchPage{Query: query, Page: page}
                },
                TypeID: SearchPage_TypeID,
            },
        },
    },
})
```

### Phase 2: Optional and Wildcard Parameters

Add flexible route matching for optional segments and catch-all routes.

**Optional parameters:**
```go
Path: "/blog/{year?}/{month?}" // Matches /blog, /blog/2026, /blog/2026/11
```

**Wildcard parameters:**
```go
Path: "/files/{*filepath}" // Captures remaining path: /files/docs/manual.pdf
```

### Phase 3: Parameter Constraints and Validation

Add type constraints and regex validation for route parameters.

```go
Path: "/users/{id:int}"                    // Only matches numeric IDs
Path: "/posts/{slug:regex([a-z0-9-]+)}"    // Pattern validation
Path: "/blog/{year:range(2000,2030)}"      // Range validation
```

### Phase 4: Navigation Guards

```go
engine.BeforeNavigate(func(from, to string) bool {
    if !user.IsAuthenticated() && isProtectedRoute(to) {
        return false  // Block navigation
    }
    return true
})
```

### Phase 4: Route Metadata

```go
routerEngine.RegisterRoutes([]router.Route{
    {
        Path: "/admin",
        Chain: adminChain,
        Meta: map[string]any{
            "requiresAuth": true,
            "title": "Admin Panel",
        },
    },
})
```

### Phase 5: Lazy Loading

```go
routerEngine.RegisterRoutes([]router.Route{
    {
        Path: "/admin",
        Chain: []router.ComponentMetadata{
            {
                Factory: func(params map[string]string) runtime.Component {
                    // Load admin module on demand
                    return loadAdminModule()
                },
                TypeID: AdminModule_TypeID,
            },
        },
    },
})
```

---

## Performance Considerations

### VDOM Patching with Event Listeners

- **Cloning overhead**: Minimal in modern browsers (~1-2ms for typical elements)
- **Trade-off**: Slight performance cost for correctness and simplicity
- **Optimization**: Only clone when `hasEventHandlers` is true

### Route Matching

- **Algorithm**: O(n) where n = number of registered routes (linear search through routes map)
- **Typical usage**: Small number of routes (< 50), negligible impact
- **Future optimization**: Trie-based matching for large route tables

### Component Instance Preservation (Pivot Algorithm)

- **Strategy**: Calculate pivot point where route chains diverge by TypeID
- **Benefit**: Layouts before pivot are reused, maintaining state and avoiding re-initialization
- **Performance**: O(min(currentChain.length, targetChain.length)) comparison, typically O(1) to O(3)
- **Memory**: Only components after pivot are recreated; preserved instances are just pointer copies

---

## Conclusion

The No-JS framework's Router Engine achieves sophisticated routing with layout management:

✅ **Pluggable**: NavigationManager interface allows alternative router implementations  
✅ **Layout-Aware**: Pivot algorithm preserves layout state across navigations  
✅ **Integrated**: Seamless VDOM and lifecycle integration with AppShell pattern  
✅ **Correct**: Proper event listener cleanup prevents bugs  
✅ **Efficient**: Minimal component recreation and scoped VDOM updates  
✅ **Developer-Friendly**: Type-safe API with compile-time TypeIDs  

The Router Engine handles the complexities of browser APIs, layout hierarchies, component lifecycle, event management, and VDOM patching while exposing a clean, type-safe API to framework users.

For detailed information about the pivot algorithm, layout chains, AppShell pattern, and memory management, see [ROUTER_ENGINE_LAYOUTS.md](ROUTER_ENGINE_LAYOUTS.md).
