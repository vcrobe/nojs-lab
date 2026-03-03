# nojs

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-MVP%2FExperimental-yellow)](https://github.com/ForgeLogic/nojs)

### The Type-Safe Web Framework for Go Purists

**nojs** is a high-performance, AOT-compiled web framework that allows you to build sophisticated front-end applications entirely in **Go**. By treating the Go compiler as your "Inspector," nojs eliminates entire classes of runtime errors through strict type-safety and a familiar, idiomatic developer experience.

> ⚠️ **Project Status:** This is an MVP (Minimum Viable Product) and is **not production-ready**. The API may change, and some features are still in active development. Use for experimentation and feedback.

---

## ✨ Key Features

- **🔒 Type-Safe Templates** - Compile-time validation of props, methods, and expressions
- **⚡ Virtual DOM** - Efficient diffing and patching for minimal DOM operations
- **🧩 Component Model** - Reusable UI building blocks with Go structs and HTML templates
- **🗺️ Built-in Router** - SPA routing with layout support and nested routes
- **📋 Optimized Lists** - `trackBy` keys for efficient list updates and re-rendering
- **🎯 Event Handling** - Type-safe event binding validated at compile time
- **🚀 WebAssembly** - Near-native performance in the browser via Go WASM
- **🛠️ AOT Compilation** - Template parsing at build time, not runtime

---

## 🏗️ Core Philosophy

Every feature in **nojs** is guided by principles designed for developers who value stability and performance over "magic."

- **Type Safety Above All:** We prefer compile-time safety over runtime flexibility. Our AOT compiler validates props, methods, and expressions before they ever reach the browser.
    
- **Go-Idiomatic by Default:** Templates and component architectures feel like natural extensions of Go. Syntax patterns (like `for...range`) are modeled directly on Go semantics.
    
- **Explicit > Implicit:** We favor clarity and control. Features like manual state updates (`StateHasChanged()`) and mandatory list keys (`trackBy`) make data flow predictable and easy to debug.
    
- **Simplicity Through Focus:** A lean, focused API that avoids complexity for little practical benefit.
    
- **Unopinionated Foundation:** We provide the core (rendering, lifecycle, type-safety) but leave state management and project structure to you.

> For the full principles behind these decisions, read the [NoJS Manifesto](https://forgelogic.github.io/nojs/design/manifesto/).

---

## 🎮 Live Demo

Want to see **nojs** in action? Check out the interactive demo at [forgelogic.github.io/nojs/demo](https://forgelogic.github.io/nojs/demo).

The demo showcases core framework features—components, routing, event handling, and list rendering—all running entirely in Go-compiled WebAssembly.

---

## 🚀 Getting Started

### Prerequisites

- Go 1.25+
- Make
- Python 3 (required for `make serve`, or use any static file server of your choice)

---

### Running the demo app

For detailed instructions on running the demo app, see the [Getting Started Guide](https://forgelogic.github.io/nojs/getting-started/).

---

### Starting your own project

See the **[Installation Guide](https://forgelogic.github.io/nojs/installation/)** for a step-by-step guide to creating a new project using the demo app as a scaffold.

---

### Repository Structure

The repository is organized as a Go workspace:

- **`nojs/`** — Core framework code (runtime, VDOM)
- **`compiler/`** — AOT template compiler (`nojsc`), maintained as a separate module
- **`router/`** — SPA routing engine, maintained as a separate module
- **`app/`** — Example application demonstrating framework features (also the recommended project scaffold)
- **`go.work`** — Workspace configuration linking all modules

The framework (`github.com/ForgeLogic/nojs`), compiler (`github.com/ForgeLogic/nojs/compiler`), router (`github.com/ForgeLogic/nojs/router`), and app (`github.com/ForgeLogic/app`) are separate modules that work together during development.

> **Note:** The compiler, router, and demo app are currently co-located in this repository for convenience during active development. Once each reaches a stable API, it will be moved to its own dedicated repository.

### Build commands

| **Command**      | **Action**                                       |
| ---------------- | ------------------------------------------------ |
| `make full`      | Compiles templates + WASM (Development)          |
| `make full-prod` | Compiles templates + WASM (Production/Optimized) |
| `make wasm`      | Rebuilds WASM only (Fast: ~1-2 seconds)          |
| `make serve`     | Starts development server on port 9090           |
| `make clean`     | Removes generated WASM binaries                  |

> **Tip:** Enable **"Disable cache"** in your browser's DevTools Network tab to ensure the WASM module reloads on every refresh.

---

## 🛠️ The AOT Compiler

The heart of **nojs** is its Ahead-of-Time compiler. It transforms declarative HTML templates (`.gt.html`) into high-performance Go component code.

### How It Works

The compiler scans directories for `*.gt.html` files and auto-generates corresponding `.generated.go` files with `Render()` methods.

### Development Workflow

1. **Create your component:**
   - `MyComponent.gt.html` - HTML template with Go expressions
   - `mycomponent.go` - Go struct with state and event handlers

2. **Compile templates:**
    
    Bash
    
    ```
    # Compile all templates in the components directory
    make full
    
    # Or run the compiler directly
    go run github.com/ForgeLogic/nojs/cmd/nojs-compiler -in ./app/internal/app/components
    ```
    
    This generates `MyComponent.generated.go` with the `Render()` method.

3. **Use the component:** Import and instantiate your component like any Go struct. The framework handles rendering, diffing, and updates.

### CLI Options

- **`-in <directory>`** - Source directory to scan for `*.gt.html` files
- **`-dev`** - Enable development mode (verbose errors, warnings)

---

## 📚 Quick Example

Here's a simple counter component to illustrate the framework:

**counter.go:**
```go
package components

import "github.com/ForgeLogic/nojs/runtime"

type Counter struct {
    runtime.ComponentBase
    Count int
}

func (c *Counter) Increment() {
    c.Count++
    c.StateHasChanged() // Trigger re-render
}

func (c *Counter) Decrement() {
    c.Count--
    c.StateHasChanged() // Trigger re-render
}
```

**Counter.gt.html:**
```html
<div class="counter">
    <h2>Count: {c.Count}</h2>
    <button @onclick="Increment">+</button>
    <button @onclick="Decrement">-</button>
</div>
```

---

## 📖 Documentation

Full documentation is available at **[forgelogic.github.io/nojs](https://forgelogic.github.io/nojs/)**.

- **[Installation Guide](https://forgelogic.github.io/nojs/installation/)** — How to create a new project using nojs
- **[Quick Guide](https://forgelogic.github.io/nojs/guides/quick-guide/)** — Practical reference for all framework features
- **[NoJS Manifesto](https://forgelogic.github.io/nojs/design/manifesto/)** — The principles and philosophy behind every design decision
- **[Design Decisions](https://forgelogic.github.io/nojs/design/design-decisions/)** — Why certain constraints exist and what trade-offs were made
- **[List Rendering](https://forgelogic.github.io/nojs/guides/list-rendering/)** — Working with dynamic lists and `trackBy` optimization
- **[Router Architecture](https://forgelogic.github.io/nojs/architecture/router-architecture/)** — SPA routing and navigation
- **[Router Layouts](https://forgelogic.github.io/nojs/architecture/router-layouts/)** — Nested layouts and content projection
- **[Inline Conditionals](https://forgelogic.github.io/nojs/guides/inline-conditionals/)** — Conditional rendering in templates
- **[Text Node Rendering](https://forgelogic.github.io/nojs/guides/text-node-rendering/)** — How text content is processed

---

## 🤝 Contributing

**nojs** is an open-source project. We welcome contributions from developers who share our passion for type-safe, performant web tools.

1. Fork the repo.
    
2. Create your feature branch.
    
3. Ensure your code follows Go-idiomatic patterns.
    
4. Submit a Pull Request.
    

---

## 📜 License

Distributed under the **Apache License 2.0**. See `LICENSE` for more information.