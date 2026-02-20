# nojs Framework — Quick Guide

A practical reference for using each implemented feature. Examples show the minimal, idiomatic usage pattern for each area.

---

## 1. Core Runtime

### Defining a Component

Every component must implement the `Component` interface: a `Render(Renderer) *vdom.VNode` method and embed `ComponentBase` (which provides `SetRenderer`).

> **Prefer templates over hand-written `Render()` methods.** In normal usage you should never write `Render()` by hand. Instead, create a `MyComponent.gt.html` template alongside your Go struct and let the AOT compiler generate the `Render()` method for you (see [Section 6 — AOT Compiler](#6-aot-compiler)). Hand-writing `Render()` is only necessary for low-level framework internals or components with logic that cannot be expressed in a template (e.g., `AppShell`, `RouterLink`).

The minimal struct — with or without a template — always looks the same:

```go
// mycomponent.go
package mypackage

import (
    "github.com/vcrobe/nojs/runtime"
)

type MyComponent struct {
    runtime.ComponentBase
    Title string // exported = prop (passed in by parent)
    count int    // unexported = private state
}
```

Pair it with a template file and the compiler generates `Render()` for you:

```html
<!-- MyComponent.gt.html -->
<div>
    <p>{Title}</p>
</div>
```

The compiler produces `MyComponent.generated.go` with the `Render()` method — you never write or edit it directly.

If you are writing `Render()` by hand (discouraged for normal components):

```go
import "github.com/vcrobe/nojs/vdom"

func (c *MyComponent) Render(r runtime.Renderer) *vdom.VNode {
    return vdom.Div(nil, vdom.NewVNode("p", nil, nil, c.Title))
}
```

### StateHasChanged

Call `StateHasChanged()` after mutating component state to trigger a re-render. Inherited from `ComponentBase`.

```go
func (c *MyComponent) Increment() {
    c.count++
    c.StateHasChanged()
}
```

If the component is inside a layout slot, `StateHasChanged()` automatically scopes the re-render to that layout only. For root components it triggers a full re-render.

### Navigate

Call `Navigate(path)` from any component to trigger client-side routing without a page reload.

```go
func (c *MyComponent) GoHome() {
    c.Navigate("/")
}
```

### Prop Updates via ApplyProps

When child components are cached across renders, the framework calls `ApplyProps` to update their props. The AOT compiler generates this method automatically from exported fields. If writing by hand:

```go
func (c *MyComponent) ApplyProps(src runtime.Component) {
    if s, ok := src.(*MyComponent); ok {
        c.Title = s.Title
    }
}
```

### Instance Caching

Child components are reused across re-renders automatically. The renderer keys instances by parent pointer + the template-defined key so component state (e.g., form input values) is preserved between renders.

---

## 2. Component Lifecycle

Implement any combination of these interfaces on your component struct.

### OnMount — run once before first render

```go
func (c *MyComponent) OnMount() {
    // fetch initial data, start timers, etc.
    go c.loadData()
}
```

### OnParametersSet — run before every render (including first)

```go
func (c *MyComponent) OnParametersSet() {
    // detect prop changes
    if c.UserID != c.prevUserID {
        c.prevUserID = c.UserID
        go c.fetchUser()
    }
}
```

### OnUnmount — run once when removed from the tree

```go
func (c *MyComponent) OnUnmount() {
    c.cancel() // cancel goroutines, release resources
}
```

### Dev vs Prod mode

Build tags on `renderer_dev.go` / `renderer_prod.go` control panic behaviour:

- **Dev** (`make full`): panics inside lifecycle hooks propagate immediately — easier debugging.
- **Prod** (`make full-prod`): panics are recovered and logged — keeps the app alive in production.

No code changes are needed; the build system selects the mode.

---

## 3. Virtual DOM (VDOM)

### Helper Constructors

```go
vdom.Text("hello")                                    // text node
vdom.Paragraph("hello")                               // <p>hello</p>
vdom.Div(map[string]any{"class": "box"}, child1, ...) // <div class="box">
vdom.Button(map[string]any{}, "Click me")             // <button>
vdom.NewVNode("span", map[string]any{"id": "x"}, []*vdom.VNode{child}, "")
```

### Supported Elements

`#text`, `p`, `div`, `input`, `button`, `h1`–`h6`, `ul`, `ol`, `li`, `select`, `option`, `textarea`, `form`, `a`, `nav`, `span`, `section`, `article`, `header`, `footer`, `main`, `aside`

### Boolean Attributes

Pass a `bool` value in the attributes map; the renderer handles the correct HTML boolean rendering.

```go
vdom.NewVNode("input", map[string]any{"disabled": true, "readonly": false}, nil, "")
```

### Mounting to the DOM

```go
// In main() or a renderer setup — render to a CSS selector
vdom.RenderToSelector("#app", myVNode)
```

---

## 4. VDOM Diffing & Patching

Patching happens automatically when `StateHasChanged()` or a navigation event triggers a re-render. Key behaviours to be aware of:

- **Attribute patching** — Only changed attributes are updated; unchanged ones are left alone.
- **ComponentKey reconciliation** — When `ComponentKey` changes (e.g., the route changes), the entire subtree is replaced and all `js.Func` callbacks are released via `deepReleaseCallbacks()`.
- **Tag replacement** — If the tag type changes (e.g., `<div>` → `<span>`), the DOM node is fully replaced.
- **Input focus preservation** — When an `<input>` is focused, its value is not patched to avoid interrupting typing.

No manual diffing API is called from user code; `StateHasChanged()` and navigation are the only entry points.

---

## 5. Event System

### Handling Events in Hand-Written Components

Attach handlers by placing an adapter on the `OnClick` (or equivalent) field of a VNode:

```go
import "github.com/vcrobe/nojs/events"

func (c *MyComponent) HandleClick(e events.ClickEventArgs) {
    e.PreventDefault()
    c.count++
    c.StateHasChanged()
}

func (c *MyComponent) Render(r runtime.Renderer) *vdom.VNode {
    btn := vdom.Button(nil, "Click me")
    btn.OnClick = events.AdaptClickEvent(c.HandleClick)
    return btn
}
```

### Adapter Functions

| Adapter | Handler Signature |
|---|---|
| `AdaptClickEvent` | `func(ClickEventArgs)` |
| `AdaptChangeEvent` | `func(ChangeEventArgs)` |
| `AdaptKeyboardEvent` | `func(KeyboardEventArgs)` |
| `AdaptMouseEvent` | `func(MouseEventArgs)` |
| `AdaptFocusEvent` | `func(FocusEventArgs)` |
| `AdaptFormEvent` | `func(FormEventArgs)` |
| `AdaptNoArgEvent` | `func()` |

### Event Arg Structs

All typed arg structs embed `EventBase`, which provides:

```go
e.PreventDefault()
e.StopPropagation()
```

`ChangeEventArgs.Value` holds the new input value. `KeyboardEventArgs.Key` holds the pressed key string.

### In Templates (AOT)

Use `@event` attributes in `.gt.html` — the compiler selects the correct adapter automatically based on the handler's parameter type:

```html
<button @onclick="HandleClick">Save</button>
<input @oninput="HandleInput" />
<form @onsubmit="HandleSubmit"></form>
```

---

## 6. AOT Compiler

The compiler reads `*.gt.html` template files alongside their Go structs and generates `*.generated.go` files containing `Render()` methods. Run it as part of the build:

```bash
make full        # compile templates + build WASM (dev mode)
make full-prod   # compile templates + build WASM (prod mode)
```

Or run the compiler directly:

```bash
go run github.com/vcrobe/nojs/cmd/nojs-compiler -in ./app/internal/app/components
```

### File Convention

```
MyComponent.gt.html        ← template
mycomponent.go             ← struct + methods (no build tags required)
MyComponent.generated.go   ← auto-generated, do not edit
```

### Data Binding

Bind component fields with `{FieldName}` in text content or attribute values:

```html
<h1>{Title}</h1>
<p>Count: {Count}</p>
<a href="{Href}">{Label}</a>
```

### Ternary Expressions

```html
<p>{IsSaving ? 'Saving...' : 'Save Changes'}</p>
<div class="msg {HasError ? 'error' : 'success'}">Status</div>
```

Negation is supported: `{!IsValid ? 'disabled' : 'enabled'}`

### Boolean Attribute Shorthand

```html
<input disabled="{IsLocked}" />
<button disabled="{!IsValid}">Submit</button>
```

### Conditional Rendering

```html
{@if IsLoggedIn}
    <p>Welcome back!</p>
{@else if IsGuest}
    <p>Browsing as guest.</p>
{@else}
    <p>Please log in.</p>
{@endif}
```

> **Important:** The condition must be a single `bool` field (or state field) on the component struct — the compiler does **not** evaluate expressions. Comparisons, function calls, and compound conditions (e.g., `Count > 0`, `len(Items) == 0`, `A && B`) are **not** supported. If you need complex logic, compute a dedicated `bool` field in your component and use that instead.
>
> ```go
> // Do this — pre-compute a named bool field
> type MyComponent struct {
>     runtime.ComponentBase
>     Items       []string
>     HasItems    bool  // set this in OnParametersSet or a method
> }
> ```
>
> ```html
> <!-- Then use the field directly -->
> {@if HasItems}
>     <ul>...</ul>
> {@endif}
> ```
>
> The compiler validates at build time that the named field exists and is of type `bool`.

### List Rendering

```html
{@for i, item := range Items trackBy item}
    <li>{i}: {item}</li>
{@endfor}
```

For slices of structs, use a field as the track key:

```html
{@for i, product := range Products trackBy product.ID}
    <li>{i}: {product.Name} (ID: {product.ID})</li>
{@endfor}
```

Both the index and value variables are required (`_` is valid for the index). The `trackBy` clause is required for correct VDOM reconciliation. Nested `{@for}` loops are supported.

### Event Binding in Templates

```html
<button @onclick="IncrementCounter">+</button>
<input @oninput="HandleInput" />
<select @onchange="HandleChange"></select>
```

The compiler validates at build time that:
- The method exists on the component struct.
- The method's parameter type matches the event (e.g., `func()`, `func(events.ClickEventArgs)`).
- The event is valid for the HTML element.

### Compile-Time Validation

The compiler reports errors for:
- Unknown field names in `{binding}` expressions.
- Non-existent event handler methods or wrong signatures.
- Unbalanced `{@for}`/`{@endfor}` and `{@if}`/`{@endif}` blocks.
- Component names that collide with standard HTML tags (e.g., use `RouterLink`, not `Link`).

---

## 7. Content Projection (Slots)

A layout component exposes a single `[]*vdom.VNode` field as its slot. The field name is irrelevant; the type is the signal.

### Defining a Layout with a Slot

```go
// mainlayout.go
type MainLayout struct {
    runtime.ComponentBase
    BodyContent []*vdom.VNode // this field is the slot
}
```

In the template, render the slot with `{BodyContent}`:

```html
<!-- MainLayout.gt.html -->
<div>
    <header><h1>My App</h1></header>
    <main>
        {BodyContent}
    </main>
</div>
```

### Using a Layout as a Parent

The router handles slot injection automatically when a layout appears before a page component in a route chain (see Section 8).

When a page component inside a slot calls `StateHasChanged()`, the framework detects the slot relationship (tracked in Go memory via `SetSlotParent`) and triggers a scoped re-render of only the layout, not the entire app.

---

## 8. Router

### Registering Routes

```go
// routes.go
routerEngine.RegisterRoutes([]router.Route{
    {
        Path: "/",
        Chain: []router.ComponentMetadata{
            {Factory: func(p map[string]string) runtime.Component { return mainLayout }, TypeID: MainLayout_TypeID},
            {Factory: func(p map[string]string) runtime.Component { return &pages.HomePage{} }, TypeID: HomePage_TypeID},
        },
    },
    {
        Path: "/blog/{year}",
        Chain: []router.ComponentMetadata{
            {Factory: func(p map[string]string) runtime.Component { return mainLayout }, TypeID: MainLayout_TypeID},
            {Factory: func(p map[string]string) runtime.Component {
                year, _ := strconv.Atoi(p["year"])
                return &pages.BlogPage{Year: year}
            }, TypeID: BlogPage_TypeID},
        },
    },
})
```

- `Chain` lists components from outermost layout to innermost page.
- `TypeID` is a unique integer per component type, used by the pivot algorithm to detect which layouts can be reused.
- `{year}` in the path becomes a key in the `params` map.

### Wiring the Router in main()

```go
func main() {
    routerEngine := router.NewEngine(nil)
    renderer := runtime.NewRenderer(routerEngine, "#app")
    routerEngine.SetRenderer(renderer)

    registerRoutes(routerEngine, mainLayout, ctx)

    appShell := router.NewAppShell(mainLayout)
    renderer.SetCurrentComponent(appShell, "app-shell")
    renderer.ReRender()

    routerEngine.Start(func(chain []runtime.Component, key string) {
        appShell.SetPage(chain, key)
    })

    select {} // keep WASM runtime alive
}
```

### Programmatic Navigation

From any component:

```go
func (c *MyComponent) GoToAbout() {
    c.Navigate("/about")
}
```

### Layout Reuse (Pivot Algorithm)

When navigating between routes that share a layout prefix (e.g., `/` and `/about` both use `MainLayout`), the layout instance is preserved and only the page component is swapped. `OnUnmount` is called on removed components; `OnMount` is called on newly created ones.

### RouterLink Component

Use the built-in `RouterLink` component in templates for client-side navigation links:

```html
<RouterLink Href="/about">Go to About</RouterLink>
<RouterLink Href="/blog/{item}">Blog {item}</RouterLink>
```

---

## 9. Build System

```bash
make full        # compile AOT templates + build WASM + serve (dev mode, panics propagate)
make full-prod   # compile AOT templates + build WASM (prod mode, panics recovered)
make wasm        # build WASM only
make serve       # serve app/wwwroot on localhost
make clean       # remove build artifacts
```

WASM compilation uses:

```bash
GOOS=js GOARCH=wasm go build -o main.wasm
```

Dev/Prod behaviour is controlled by Go build tags: `(js || wasm) && dev` for dev, `(js || wasm) && !dev` for prod. These are set by the Makefile targets automatically.

The workspace uses a `go.work` file linking the `nojs` framework module, the `compiler` module, and the `app` example module so they can all reference each other locally without publishing to a module proxy.

---

## 10. JS ↔ Go Interop

### Exporting a Go Function to JavaScript

```go
// main.go
js.Global().Set("add", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
    return args[0].Int() + args[1].Int()
}))
```

Call from the browser console: `window.add(2, 3)` → `5`

### Calling a JavaScript Function from Go

```go
js.Global().Call("myJsFunction", "arg1", 42)
```

### Keeping the WASM Runtime Alive

The `main()` function must block after setup or the WASM binary exits:

```go
func main() {
    // ... setup ...
    select {}
}
```

### Browser API Wrappers

Prefer the provided wrapper packages over raw `syscall/js`:

```go
import "github.com/vcrobe/nojs/console"
import "github.com/vcrobe/nojs/dialogs"
import "github.com/vcrobe/nojs/sessionStorage"

console.Log("value:", 42)
console.Warn("watch out")
console.Error("something failed")

dialogs.Alert("Hello!")
name := dialogs.Prompt("Your name?")

sessionStorage.SetItem("token", "abc123")
token := sessionStorage.GetItem("token")
sessionStorage.RemoveItem("token")
```

### wasm_exec.js and core.js

- `wasm_exec.js` is the vendored Go WASM runtime bridge. Keep it in sync with the Go toolchain version when upgrading Go.
- `core.js` loads `main.wasm`, instantiates the Go runtime, and bootstraps the framework. Do not call exported Go functions before `core.js` has completed `go.run(...)`.
