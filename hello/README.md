# Hello - nojs Framework Standalone Application

A minimal, stand-alone Go + WebAssembly application that demonstrates the core concepts of the nojs framework.

## Overview

This is a simple "Hello World" application built with:
- **Go + WebAssembly**: Compiles to WASM binary for browser execution
- **Virtual DOM**: Uses the nojs vdom package for efficient DOM rendering  
- **Component Model**: Demonstrates a reusable component pattern (HelloWorld component)
- **Framework Integration**: References the nojs framework without coupling

## Project Structure

```
hello/
├── go.mod                   # Go module definition (references nojs)
├── go.sum                   # Go dependencies
├── main.go                  # WASM entrypoint (//go:build js || wasm)
└── wwwroot/                 # Web assets
    ├── index.html          # HTML shell
    ├── style.css           # Application styling
    ├── core.js             # Framework bootstrap and event handling
    └── wasm_exec.js        # Go runtime bridge (vendored from Go toolchain)
```

## Building

Build the WebAssembly binary from the `hello` directory:

```bash
cd hello
GOOS=js GOARCH=wasm go build -o wwwroot/main.wasm
```

Or use the workspace's build task if configured.

## Running

1. **Build the WASM binary** (see above)
2. **Serve the wwwroot directory** using any static HTTP server:
   ```bash
   cd wwwroot
   python3 -m http.server 9090
   ```
3. **Open in browser**: `http://localhost:9090`

You should see a purple gradient with a centered white card displaying:
- "Hello from Go + WebAssembly"
- A click counter
- An interactive button

Clicking the button increments the counter and logs the current count to the browser console.

## How It Works

### Component Architecture

The `HelloWorld` struct implements a simple component:

```go
type HelloWorld struct {
    count int
}

func (h *HelloWorld) Render(r runtime.Renderer) *vdom.VNode {
    // Returns virtual DOM tree representing the UI
}
```

### Virtual DOM Rendering

The `Render()` method returns a `vdom.VNode` tree describing the UI structure. The renderer then:
1. Converts the virtual DOM to real DOM elements
2. Mounts them into the `#app` selector
3. Handles updates when component state changes

### JS Interop

The `incrementClick` function is exposed to JavaScript, allowing the HTML button to call back into Go:

```go
func incrementClick(this any, args []any) any {
    if helloInstance != nil {
        helloInstance.IncrementCount()
    }
    return nil
}
```

The button's `onclick` handler calls `_onIncrementClick`, which is defined in `core.js` and bridges to the Go function.

## Dependencies

- `github.com/ForgeLogic/nojs`: Core framework (virtual DOM, runtime, console wrappers)
- Go 1.25.1 or higher
- Standard library only (no external dependencies beyond nojs)

## Architecture Insights

This application demonstrates:
- **Separation of concerns**: Go logic separate from HTML structure
- **Component isolation**: HelloWorld is self-contained and reusable
- **Framework abstraction**: Uses nojs interfaces without tight coupling
- **Minimal dependencies**: No CSS framework, bundler, or build tools required

## Extending the Application

To add more components:

1. Create a new struct with a `Render()` method
2. Return a `vdom.VNode` describing its UI
3. Set it on the renderer with `renderer.SetCurrentComponent()`
4. Export any event handlers needed via `js.Global().Set()`

Example:

```go
type Counter struct {
    count int
}

func (c *Counter) Render(r runtime.Renderer) *vdom.VNode {
    // Build and return VNode tree
}
```

## Notes

- All Go files use `//go:build js || wasm` to compile only for WASM targets
- The nojs framework types (`vdom.VNode`, `runtime.Renderer`, etc.) are build-tag-free (dev + wasm)
- Uses relative path replacement in `go.mod` to reference the local nojs framework
- `wasm_exec.js` is the Go runtime bridge—keep it in sync with your Go version

## Further Reading

- [nojs Documentation](../nojs/documentation/)
- [Virtual DOM Design](../nojs/documentation/DESIGN_DECISIONS.md)
- [Component Model](../nojs/documentation/QUICK_GUIDE.md)
