# Copilot instructions for this repo

Purpose: Help AI agents work productively in this Go + WebAssembly (WASM) framework without guessing. Keep answers concrete, repo-specific, and runnable.

## Big picture
- This is a Go-based client-side web framework using WebAssembly for near-native performance in the browser.

- The framework is built on key pillars from modern frameworks like React, Vue, and Blazor, adapted for the Go ecosystem:
  - **Go + WASM**: Go code compiles to WASM binary for browser execution via syscall/js
  - **Component Model**: Reusable UI building blocks combining Go structs (logic/state) with HTML templates (structure)
  - **Virtual DOM**: In-memory DOM representation with efficient diff/patch cycles to minimize real DOM operations
  - **AOT Compilation**: Ahead-of-Time template parsing that auto-generates Render() methods from HTML templates

- Current state: Basic WASM interop established with minimal VDOM scaffold (`vdom/`, `component/`) supporting `<p>`, `<button>`, `<div>` and `<input>` elements.

Key files
- `main.go`: WASM entrypoint. Exports Go functions, calls into JS, then blocks with `select {}` to keep runtime alive.
- `core.js`: loads `main.wasm` with `Go` from `wasm_exec.js`, provides component loading and framework bootstrapping.
- `labtests.js`: browser-side helpers for testing Go-JS interop.
- `console/`, `dialogs/`, `sessionStorage/`: thin wrappers around `syscall/js` for browser APIs.
- `vdom/vnode.go`: VNode struct representing virtual DOM nodes with type, attributes, children, and text content.
- `vdom/render.go`: minimal DOM renderer implementing the patch process (currently supports `<p>`, `<button>`, `<div>`, `<input>`).
- `component/component.go`: Component interface defining `Render() *vdom.VNode` for UI building blocks.
- `index.html`: static HTML shell for the application.

Future architecture (per blueprint):
- `compiler/`: AOT template parser and code generator for HTML-to-Go Render() methods
- `core/`: framework engine with component lifecycle, scheduling, and update management
- `router/`: SPA routing logic for single-page application navigation
- `stdlib/`: standard library of built-in components

The framework roadmap follows three key phases:
1. **WASM Hello World**: Basic Go-to-WASM interop with DOM manipulation
2. **Basic VDOM**: Manual VNode creation with simple rendering (current state)
3. **AOT Compiler**: HTML template parsing and automatic Render() method generation

## Build and run (local dev)
- Build wasm at repo root:
  - Environment: `GOOS=js GOARCH=wasm`
  - Command: `go build -o main.wasm`
  - Output: `main.wasm` binary for browser execution
- Serve the project folder (any static server) and open `index.html`.

Example development cycle:
1) Build: `GOOS=js GOARCH=wasm go build -o main.wasm`
2) Serve: `python3 -m http.server 9090` (or any static server)
3) Browse: `http://localhost:9090` (console logs show module load)
4) Test: Use browser DevTools to interact with exported Go functions

## Framework architecture and data flow
The framework follows a modern component-based architecture with virtual DOM:

### Component Model
Components are reusable UI building blocks that combine:
- **Go structs**: Handle logic, state management, and event handlers
- **HTML templates**: Define visual structure (future AOT compilation target)
- **Render() method**: Returns `*vdom.VNode` representing component's UI

Example component pattern:
```go
type Counter struct {
    count int
}

func (c *Counter) Increment() {
    c.count++
    // Framework triggers re-render
}

func (c *Counter) Render() *vdom.VNode {
    // Currently hand-written, future: auto-generated from HTML template
    return vdom.NewVNode("div", nil, []*vdom.VNode{
        vdom.NewVNode("p", nil, nil, fmt.Sprintf("Count: %d", c.count)),
        vdom.NewVNode("button", map[string]string{"onclick": "increment"}, nil, "Click Me"),
    }, "")
}
```

### Virtual DOM (VDOM) Lifecycle
1. **Render**: Component's `Render()` method creates new VDOM tree
2. **Diff**: New VDOM compared against previous VDOM tree  
3. **Patch**: Minimal set of DOM changes calculated
4. **Apply**: Changes applied to real DOM via `syscall/js`

This minimizes expensive DOM operations and maximizes performance.

### Template Compilation (Future)
HTML templates will be compiled Ahead-of-Time (AOT):
- Parse `*.component.html` files with Go template syntax
- Generate corresponding `Render()` methods automatically  
- Support event binding (`go-on:click`) and data binding (`{c.count}`)
- All parsing happens at build time, not runtime

### Application Data Flow
The framework follows this update lifecycle:
1. **User Action**: Events (button clicks, input changes) trigger event listeners
2. **Component Logic**: Go event handlers update component state
3. **Re-render**: Framework calls component's `Render()` method to create new VDOM tree
4. **Diffing**: New VDOM tree is compared against previous VDOM tree
5. **Patching**: Minimal set of DOM changes is calculated and applied via `syscall/js`

This approach minimizes expensive DOM operations while maintaining reactive updates.

Notes
- `wasm_exec.js` is vendored (Go runtime bridge). Keep in sync with installed Go when upgrading.
- `core.js` expects `main.wasm` at project root.

## Patterns and conventions
- Build tags: All browser-facing Go files use `//go:build js || wasm` to target wasm.
- JS↔Go interop:
  - Export Go to JS by `js.Global().Set("name", js.FuncOf(fn))` (see `add` in `main.go`).
  - Call JS from Go via `js.Global().Call("fnName", args...)` (see `calledFromGoWasm`).
  - Keep Go alive with `select {}` at end of `main()`.
- Browser API wrappers: Prefer packages `console`, `dialogs`, `sessionStorage` over raw `syscall/js` in app code.
- HTML partials: Loaded into `#header`, `#content`, `#footer` with `loadComponent()`; keep paths relative to repo root when serving.
- VDOM: `vdom.VNode` has a minimal renderer; no diff/patch logic. Only `<p>` renders; others are ignored for now.

## Commit message guidelines
- When suggesting a commit message, always use the [Conventional Commits](https://www.conventionalcommits.org/) specification (e.g., `feat:`, `fix:`, `chore:`, etc.).
- The commit message must include very detailed information about the reason for the changes, not just what was changed.
- Example: `fix(component): correct VNode rendering for <p> elements to prevent double rendering in vdom/render.go. This resolves a bug where paragraphs were duplicated due to incorrect child node handling.`

## Adding features (examples)
- Expose a new Go function to JS:
  - Implement `func doThing(this js.Value, args []js.Value) interface{}` in `main.go` or a new file with wasm build tag.
  - Register with `js.Global().Set("doThing", js.FuncOf(doThing))`.
  - Call from JS: `window.doThing("arg")` after wasm has started (after core.js runs `go.run`).
- Use wrappers:
  - Logs: `console.Log("msg", 123)`; Warn/Error similar.
  - Dialogs: `dialogs.Alert("Hi")`, `name := dialogs.Prompt("Your name?")`.
  - Session storage: `sessionStorage.SetItem("k","v")`, `GetItem`, `RemoveItem`, `Clear`, `Length`.
- Compose VNodes:
  - Create nodes: `vdom.NewVNode("p", nil, nil, "hello")` or `vdom.Paragraph("hello")`
  - Mount: `vdom.RenderToSelector("#content", vdom.Paragraph("Hi"))`
  - Implement component: type MyComp struct{}; func (c MyComp) Render() *vdom.VNode { return vdom.NewVNode("div", nil, nil, "hi") }

## Debugging tips
- Open browser DevTools:
  - Console should print "WebAssembly module loaded." and logs from `main.go` calls.
  - If `add` is undefined, ensure `main.go` exported it and wasm is rebuilt/served fresh.
- Common pitfalls:
  - Not rebuilding after Go changes (always rebuild `main.wasm`).
  - Serving from wrong directory or missing `wasm_exec.js`/`core.js` includes in `index.html`.
  - Calling JS before `go.run(...)` completes; wait until the wasm runtime has started.

## Upgrades and compatibility
- Go version is declared in `go.mod`. If you upgrade Go, refresh `wasm_exec.js` to the matching version (from your Go toolchain).
- Keep build tags consistent across browser-targeted packages.

## Quick references
- Entrypoint: `main.go`
- Interop: syscall/js + wrappers in `console/`, `dialogs/`, `sessionStorage/`
- UI: `index.html`, `core.js`, `labtests.js`
- VDOM types: `vdom/vnode.go`, API contract: `component/component.go`
