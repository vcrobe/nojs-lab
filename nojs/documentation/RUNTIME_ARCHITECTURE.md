# Runtime Architecture

This document describes every type and subsystem in the `nojs/runtime` package: what it does, how the pieces connect, and how the rendering lifecycle progresses from component definition to DOM update.

---

## Table of Contents

1. [Package overview](#1-package-overview)
2. [Build tag strategy](#2-build-tag-strategy)
3. [Core interfaces](#3-core-interfaces)
   - [Component](#component)
   - [Renderer](#renderer)
   - [NavigationManager](#navigationmanager)
   - [Navigator](#navigator)
4. [ComponentBase — embedded state helpers](#4-componentbase--embedded-state-helpers)
5. [Lifecycle interfaces](#5-lifecycle-interfaces)
   - [Mountable](#mountable)
   - [ParameterReceiver](#parameterreceiver)
   - [Unmountable](#unmountable)
   - [PropUpdater](#propupdater)
6. [RendererImpl — the concrete renderer](#6-rendererimpl--the-concrete-renderer)
   - [Internal state](#internal-state)
   - [Construction](#construction)
   - [RenderRoot](#renderroot)
   - [RenderChild](#renderchild)
   - [ReRender and ReRenderSlot](#rerender-and-rerenderslot)
   - [Component cleanup](#component-cleanup)
   - [Navigate](#navigate)
7. [Dev vs. production lifecycle dispatch](#7-dev-vs-production-lifecycle-dispatch)
8. [Full render lifecycle walkthrough](#8-full-render-lifecycle-walkthrough)
9. [Slot / layout scoped re-renders](#9-slot--layout-scoped-re-renders)
10. [Thread safety](#10-thread-safety)
11. [File map](#11-file-map)

---

## 1. Package overview

The `runtime` package is the **execution engine** of nojs. It owns:

- The `Component` and `Renderer` interfaces that all components and the renderer must satisfy.
- `ComponentBase` — a composable struct that every component embeds to receive `StateHasChanged()`, `Navigate()`, and slot-parent tracking for free.
- Lifecycle interfaces (`Mountable`, `ParameterReceiver`, `Unmountable`, `PropUpdater`) that components opt into.
- `RendererImpl` — the concrete WASM-only renderer that manages the component instance tree, drives the virtual DOM lifecycle (initial render, patch, clear), and wires up client-side navigation.

> **Signals are out of scope for this package.** Cross-component and persistent application state is handled by `github.com/ForgeLogic/nojs/signals` (`Signal[T]`), which has no dependency on the runtime, VDOM, or renderer. See the [Signals documentation](./SIGNALS.md) for details.

The package is deliberately split between build-tag-free files (interfaces / `ComponentBase`) and WASM-only files (`RendererImpl`, lifecycle dispatch) so that compiler-generated `Render()` methods and tests can be compiled on native platforms without a WASM toolchain.

---

## 2. Build tag strategy

| File | Build constraint | Purpose |
|---|---|---|
| `component.go` | none | `Component` interface + `ComponentFactory` |
| `componentbase.go` | none | `ComponentBase` struct |
| `componentlifecycle.go` | `js \|\| wasm` | Lifecycle interfaces (`Mountable`, etc.) |
| `navigation.go` | `js && wasm` | `NavigationManager` + `Navigator` interfaces |
| `renderer.go` | none | `Renderer` interface |
| `renderer_impl.go` | `js \|\| wasm` | Concrete `RendererImpl` |
| `renderer_dev.go` | `(js \|\| wasm) && dev` | Lifecycle dispatch — dev mode (panics propagate) |
| `renderer_prod.go` | `(js \|\| wasm) && !dev` | Lifecycle dispatch — prod mode (panics recovered) |

Files with **no build tag** can be imported by native Go test binaries. This keeps the AOT-generated `Render()` methods and their unit tests fully buildable without a WASM target.

---

## 3. Core interfaces

### Component

```go
// component.go
type Component interface {
    Render(r Renderer) *vdom.VNode
    SetRenderer(r Renderer)
}
```

Every UI building block implements `Component`. The framework calls `SetRenderer` during mounting so the component can later trigger re-renders via `StateHasChanged()`. `Render` returns the complete virtual DOM subtree for that component on every render cycle.

`ComponentFactory` is related:

```go
type ComponentFactory func(params map[string]string) Component
```

The router uses `ComponentFactory` to instantiate components for matched routes, passing URL path parameters (e.g., `{id}` → `"42"`) as the `params` map.

---

### Renderer

```go
// renderer.go
type Renderer interface {
    RenderChild(key string, childWithProps Component) *vdom.VNode
    ReRender()
    ReRenderSlot(slotParent Component) error
    Navigate(path string) error
}
```

AOT-generated `Render()` methods call `r.RenderChild(...)` for every child component they embed. The `Renderer` interface has no build tags, which is what makes generated code cross-platform.

| Method | Called by | Purpose |
|---|---|---|
| `RenderChild` | Generated `Render()` methods | Create/reuse a child component instance and return its VNode |
| `ReRender` | `ComponentBase.StateHasChanged()` | Full re-render from the root |
| `ReRenderSlot` | `ComponentBase.StateHasChanged()` (slot path) | Scoped re-render of a layout's slot content |
| `Navigate` | `ComponentBase.Navigate()` | Delegate to the router |

---

### NavigationManager

```go
// navigation.go (js && wasm only)
type NavigationManager interface {
    Start(onChange func(chain []Component, key string)) error
    Navigate(path string) error
    GetComponentForPath(path string) (Component, bool)
}
```

The framework core is **router-agnostic**. Any router that satisfies `NavigationManager` can be injected into `RendererImpl` at construction time. `nil` is a valid value — the renderer operates without routing.

- `Start` — reads the initial browser URL, resolves the component chain, fires `onChange`, then begins listening to `popstate` events.
- `Navigate` — pushes a new entry onto the browser history stack and fires `onChange`.
- `GetComponentForPath` — resolves a path to a component without side effects (used for pre-resolution).

The `onChange` callback receives a **chain** (`[]Component`) rather than a single component because a routed app may compose a layout, a sub-layout, and a leaf page all in one navigation.

---

### Navigator

```go
// navigation.go (js && wasm only)
type Navigator interface {
    Navigate(path string) error
}
```

A narrower interface used when only navigation is needed (i.e., from components). `ComponentBase` satisfies this interface through its `Navigate` method.

---

## 4. ComponentBase — embedded state helpers

```go
// componentbase.go (no build tag)
type ComponentBase struct {
    renderer   Renderer
    slotParent Component
}
```

Embed `ComponentBase` in every component struct:

```go
type Counter struct {
    runtime.ComponentBase
    Count int
}
```

Embedding provides:

| Method | Signature | Purpose |
|---|---|---|
| `SetRenderer` | `(r Renderer)` | Injected by the framework; stores the renderer for later use |
| `GetRenderer` | `() Renderer` | Returns the stored renderer (for advanced use cases) |
| `StateHasChanged` | `()` | Signal that state mutated; triggers a re-render |
| `SetSlotParent` | `(parent Component)` | Called by the renderer when this component is slotted inside a layout |
| `Navigate` | `(path string) error` | Request client-side navigation via the router |

### StateHasChanged routing logic

```
ComponentBase.StateHasChanged()
    │
    ├─ slotParent != nil ──► renderer.ReRenderSlot(slotParent)   // scoped re-render
    │
    └─ slotParent == nil ──► renderer.ReRender()                 // full re-render
```

This means page components that live inside a layout's `[]*vdom.VNode` slot trigger only a slot re-render — the layout shell is diffed but not recreated.

---

## 5. Lifecycle interfaces

All lifecycle interfaces are declared in `componentlifecycle.go` (build tag: `js || wasm`).

### Mountable

```go
type Mountable interface {
    OnMount()
}
```

Called **once**, before the component's first render, after its instance is created. Use it for one-time initialization such as kicking off async data fetches.

```go
func (c *UserProfile) OnMount() {
    c.IsLoading = true
    go c.fetchUserData() // goroutine calls StateHasChanged when done
}
```

### ParameterReceiver

```go
type ParameterReceiver interface {
    OnParametersSet()
}
```

Called **before every render**, including the first one. Useful for detecting prop changes and reacting to them (e.g., fetching new data when a `DataID` prop changes).

```go
func (c *DataDisplay) OnParametersSet() {
    if c.DataID != c.prevDataID {
        c.prevDataID = c.DataID
        go c.fetchData()
    }
}
```

### Unmountable

```go
type Unmountable interface {
    OnUnmount()
}
```

Called **once** when the component instance is removed from the active tree (e.g., when a route changes away from the component). Use it to cancel goroutines, clear intervals, or release any held resources.

```go
func (c *TimerComponent) OnUnmount() {
    if c.cancel != nil {
        c.cancel() // stops the ticker goroutine
    }
}
```

### PropUpdater

```go
type PropUpdater interface {
    ApplyProps(source Component)
}
```

**Do not implement this manually.** The AOT compiler generates `ApplyProps` for every component. The renderer calls it when a component instance already exists in the cache and new props arrive from the parent — it copies exported (prop) fields while leaving unexported (state) fields untouched.

---

## 6. RendererImpl — the concrete renderer

`RendererImpl` is declared in `renderer_impl.go` and is only available in WASM builds (`js || wasm`).

### Internal state

```go
type RendererImpl struct {
    mu                sync.Mutex
    instances         map[string]Component          // keyed component instance cache
    initialized       map[string]bool               // tracks which instances have been OnMount'd
    activeKeys        map[string]bool               // components active in the current render cycle
    currentComponent  Component                     // the root component
    currentKey        string                        // reconciliation key (e.g., current route path)
    navManager        NavigationManager             // optional router
    mountID           string                        // CSS selector for the DOM mount point
    prevVDOM          *vdom.VNode                   // VDOM from the previous render cycle
    instanceVDOMCache map[Component]*vdom.VNode     // per-instance VDOM (for slot diffs)
    renderingStack    []Component                   // stack of currently-rendering components
}
```

Key design decisions:

- **`instances` map** — keyed by a globally unique string built from the parent component pointer and the child's logical key (e.g., `"0xc000123456:counter-0"`). This prevents key collisions when multiple parent components render children with the same logical key.
- **`renderingStack`** — a call-stack maintained during `Render()` traversal; used to construct the globally unique child key during `RenderChild`.
- **`instanceVDOMCache`** — stores the last rendered VNode for each component instance, enabling `ReRenderSlot` to diff against the previous tree without a full root re-render.

### Construction

```go
renderer := runtime.NewRenderer(navManager, "#app")
```

`navManager` may be `nil` for apps without routing. `mountID` is a CSS selector for the DOM element that acts as the application root (e.g., `"#app"`).

### RenderRoot

Called once at application startup and on every full re-render (`ReRender`):

1. Resets `activeKeys` map.
2. Calls `SetRenderer` on `r.currentComponent`.
3. Pushes root onto `renderingStack`.
4. Calls `OnMount` (first render only) then `OnParametersSet`.
5. Calls `currentComponent.Render(r)` to produce the new VDOM tree.
6. Pops root from `renderingStack`.
7. Attaches `currentKey` to the root VNode as `ComponentKey`.
8. **Initial render**: clears the mount point and calls `vdom.RenderToSelector`.
9. **Subsequent render, same key**: calls `vdom.Patch` for minimal DOM updates.
10. **Subsequent render, key changed** (navigation): clears the mount point, re-renders fresh, calls `OnUnmount` on the old root, resets `initialized`.
11. Stores `newVDOM` in `prevVDOM` and `instanceVDOMCache`.
12. Calls `cleanupUnmountedComponents`.

### RenderChild

Called by generated `Render()` methods for every embedded child component:

```go
vnode := r.RenderChild("counter-0", &Counter{Count: props.InitialCount})
```

1. Builds a globally unique key: `fmt.Sprintf("%p:%s", parent, key)`.
2. Marks the key as active (`activeKeys[globalKey] = true`).
3. **First render**: stores the provided `childWithProps` as the canonical instance.
4. **Subsequent renders**: retrieves the cached instance; calls `ApplyProps` (if `PropUpdater` is implemented) to update props without losing state.
5. Calls `SetRenderer`, then `SetSlotParent` (if applicable).
6. Calls lifecycle methods: `OnMount` (first time only), then `OnParametersSet`.
7. Pushes the instance onto `renderingStack`, calls `instance.Render(r)`, pops.
8. Returns the VNode.

### ReRender and ReRenderSlot

| Method | Scope | Mechanism |
|---|---|---|
| `ReRender()` | Full application | Delegates to `RenderRoot()` |
| `ReRenderSlot(parent)` | Single layout's slot | Re-renders the parent layout, diffs against cached VNode, patches |

`ReRenderSlot` is the performance-critical path for page components inside layouts. The algorithm:

1. Retrieve the parent layout's previous VNode from `instanceVDOMCache`.
2. Push the parent onto `renderingStack` (for consistent child key generation).
3. Call `parent.Render(r)` to produce a new VNode (the layout re-renders with updated slot content).
4. Pop from `renderingStack`.
5. Call `vdom.Patch(mountID, prevParentVDOM, newParentVDOM)`.
6. Update `instanceVDOMCache[parent]`.

If no cached VNode exists yet, `reRenderFull` is used as a fallback.

### Component cleanup

After every `RenderRoot`, `cleanupUnmountedComponents` iterates `instances`. Any instance whose key is **not** in `activeKeys` was not reached during the render walk — meaning it was removed from the tree. For such instances:

1. `OnUnmount()` is called (if the component is `Unmountable`).
2. The instance is removed from `instances` and `initialized`.

`activeKeys` is then reset for the next cycle.

### Navigate

```go
func (r *RendererImpl) Navigate(path string) error
```

Delegates to `r.navManager.Navigate(path)`. Returns an error if no router is configured.

---

## 7. Dev vs. production lifecycle dispatch

Lifecycle methods (`OnMount`, `OnParametersSet`, `OnUnmount`) are called through thin dispatcher methods so their error-handling behaviour can differ between builds.

| Build tag | File | Behaviour |
|---|---|---|
| `(js \|\| wasm) && dev` | `renderer_dev.go` | Panics propagate — crash fast for developer feedback |
| `(js \|\| wasm) && !dev` | `renderer_prod.go` | Panics are recovered and logged via `fmt.Printf` — the app keeps running |

To build for development (default framework build): pass `-tags dev`.  
To build for production: omit the `dev` tag.

---

## 8. Full render lifecycle walkthrough

Below is the sequence of events from a button click to an updated DOM.

```
User clicks button
    │
    ▼
Component event handler (e.g., Counter.Increment())
    │  mutates c.Count
    ▼
c.StateHasChanged()
    │
    ├── slotParent != nil ──► RendererImpl.ReRenderSlot(layout)
    │                              │
    │                              ├─ layout.Render(r)  ──► new VNode tree
    │                              ├─ vdom.Patch(...)   ──► DOM updated
    │                              └─ instanceVDOMCache updated
    │
    └── slotParent == nil ──► RendererImpl.RenderRoot()
                                   │
                                   ├─ OnParametersSet (if impl.)
                                   ├─ currentComponent.Render(r)
                                   │       └─ r.RenderChild("x", &Child{...})
                                   │               ├─ ApplyProps (if re-render)
                                   │               ├─ OnParametersSet (if impl.)
                                   │               └─ child.Render(r) ──► child VNode
                                   ├─ (Initial) vdom.RenderToSelector
                                   │   or
                                   │   vdom.Patch(mountID, prevVDOM, newVDOM)
                                   ├─ prevVDOM = newVDOM
                                   └─ cleanupUnmountedComponents
```

---

## 9. Slot / layout scoped re-renders

The slot system enables efficient updates when page-level state changes inside a layout shell. The mechanism is **purely in-memory** with no DOM markers.

### How slots work

1. A layout component declares a `[]*vdom.VNode` field (e.g., `BodyContent []*vdom.VNode`).
2. The router (or parent) populates this field with children VNodes.
3. When the layout is rendered, its generated `Render()` injects those children into the appropriate place in the VNode tree.
4. The AOT compiler detects the `[]*vdom.VNode` field type and generates code to inject child component VNodes into it.

### How state changes propagate

When a page component (living inside `BodyContent`) calls `StateHasChanged()`:

1. `ComponentBase` checks `b.slotParent` (set by the renderer on first slot mount).
2. If non-nil, `ReRenderSlot(b.slotParent)` is called instead of `ReRender()`.
3. `ReRenderSlot` re-renders only the layout — the layout template re-embeds the updated page VNodes.
4. `vdom.Patch` diffs old vs new layout VNode and applies only the changed DOM nodes.

This means layout chrome (navigation bars, sidebars, headers) does not touch the DOM if only page content changed.

---

## 10. Thread safety

`RendererImpl` protects all mutable state with a single `sync.Mutex` (`r.mu`). Every public method that reads or writes renderer state acquires the mutex. This allows event handlers in WASM goroutines (e.g., async data fetches calling `StateHasChanged`) to safely trigger re-renders without data races.

Lifecycle callbacks (`OnMount`, `OnParametersSet`, `OnUnmount`) are called **while the mutex is held** — avoid acquiring the same mutex inside lifecycle methods to prevent deadlocks.

---

## 11. File map

| File | Build tag | Contents |
|---|---|---|
| `component.go` | none | `Component` interface, `ComponentFactory` |
| `componentbase.go` | none | `ComponentBase` struct with `StateHasChanged`, `Navigate`, `SetSlotParent` |
| `componentlifecycle.go` | `js \|\| wasm` | `Mountable`, `ParameterReceiver`, `Unmountable`, `PropUpdater` |
| `navigation.go` | `js && wasm` | `NavigationManager`, `Navigator` |
| `renderer.go` | none | `Renderer` interface |
| `renderer_impl.go` | `js \|\| wasm` | `RendererImpl`, `NewRenderer`, full rendering engine |
| `renderer_dev.go` | `(js \|\| wasm) && dev` | `callOnMount`, `callOnParametersSet`, `callOnUnmount` — dev (panic pass-through) |
| `renderer_prod.go` | `(js \|\| wasm) && !dev` | `callOnMount`, `callOnParametersSet`, `callOnUnmount` — prod (panic recovery) |
