# Design Decisions & Trade-offs

This document explains the reasoning behind certain design choices in **nojs** that may appear unconventional or restrictive at first glance. Each section describes what the constraint is, why it exists, and what alternatives were considered.

---

## 1. No Real Compiler — Using the Go Standard Library Instead

### The decision

The `nojsc` template compiler does **not** implement a traditional compiler pipeline (lexer → parser → AST → code generator). Instead, it uses:

- **`go/packages`** — to inspect Go source files, read struct fields, and resolve exported method names
- **`golang.org/x/net/html`** — to parse `.gt.html` template files

### Why

Building a real compiler for a Go-targeting language is a significant undertaking. It would require maintaining a full grammar, a custom lexer, an AST, type resolution, and error reporting infrastructure. For an MVP-stage framework, this complexity would slow down iteration and introduce a large surface area for bugs.

The standard library approach gives us:

- Reliable, well-tested HTML parsing via `net/html`
- Full Go type inspection without re-implementing Go's type system
- A fraction of the complexity and maintenance cost

### Trade-offs

This approach imposes constraints that a real compiler would not have (see sections below). As the framework matures, a dedicated compiler with a proper AST may be introduced to lift some of these restrictions.

---

## 2. Component Template Files Must Use PascalCase Naming

### The decision

Template files **must** be named using PascalCase, matching their Go struct name exactly:

```
Counter.gt.html       ← correct
counter.gt.html       ← will not work
mycounter.gt.html     ← will not work
```

### Why

The `net/html` parser normalizes all HTML tag names to lowercase. When the compiler encounters a custom component tag such as `<Counter />` in a template, it receives it as `<counter />` — the original casing is lost.

To reconstruct the correct Go type name (which must start with an uppercase letter to be exported), the compiler uses the **template filename** as the source of truth. This is the only place in the pipeline where the original casing is reliably preserved.

### Trade-offs

- Users must be disciplined about file naming, which deviates from common Go file naming conventions (which use `snake_case` or `lowercase`).
- IDEs and file systems that are case-insensitive (e.g. macOS by default) could mask naming errors that would fail on a case-sensitive Linux system.

### Rule

The filename (without extension) must exactly match the Go struct name. For a struct named `ProductCard`, the template must be `ProductCard.gt.html`.

---

## 3. The `.gt.html` File Extension

### The decision

Template files use the `.gt.html` extension instead of plain `.html`.

### Why

Using `.html` would create ambiguity — a project's component directory could contain static HTML files, partial layouts, or other non-component HTML that should not be compiled. The `.gt.html` extension (short for **Go Template HTML**) acts as an explicit opt-in marker: only files with this extension are picked up by `nojsc`.

### Trade-offs

- Editors may not automatically apply HTML syntax highlighting to `.gt.html` files without configuration. Most editors can be configured to associate `.gt.html` with HTML syntax.

---

## 4. Go Instead of TinyGo

### The decision

The framework targets the standard **Go toolchain** (`GOOS=js GOARCH=wasm`) rather than [TinyGo](https://tinygo.org/) for compiling to WebAssembly.

### Why

TinyGo produces significantly smaller `.wasm` binaries and is an appealing target for production browser delivery. However, it achieves this by supporting only a subset of the Go language and standard library. During active framework development, these gaps would become blockers:

- Incomplete `reflect` support limits runtime introspection used by the framework internals.
- Some standard library packages (e.g. `go/packages`, `net/html`) are unavailable or partially supported.
- Certain language features (e.g. complex interface patterns, full goroutine semantics) behave differently or are unsupported.

While stabilizing the framework, having the **full Go language and standard library** at our disposal is essential. It allows us to move fast, use familiar tooling, and avoid working around TinyGo's constraints before the core model is even proven.

### Future direction

Once the framework reaches a stable, production-ready state, TinyGo will be evaluated as an alternative compilation target to reduce the final `.wasm` binary size for end users. This review will include assessing which framework features are compatible, whether any abstractions need to be adjusted, and what the real-world size and performance trade-offs look like.

### Trade-offs

- Standard Go produces larger `.wasm` binaries compared to TinyGo, which can affect initial page load time.
- This is an accepted cost during development; binary size optimisation is deferred to the production-readiness phase.

---

## 5. Explicit `StateHasChanged()` Instead of Automatic Reactivity

### The decision

Components must explicitly call `c.StateHasChanged()` after mutating state to trigger a re-render. The framework does not automatically detect state changes.

### Why

Automatic state change detection (as seen in other frameworks) requires either:
- **Runtime proxies** — wrapping every field in a proxy to intercept writes, which adds overhead and is not idiomatic in Go
- **Dirty checking** — polling or comparing state snapshots on a schedule, which is unpredictable
- **Code generation** — generating setter wrappers for every field, which increases compiler complexity significantly

Explicit `StateHasChanged()` is a deliberate design choice aligned with the framework's **Explicit > Implicit** philosophy. It makes data flow easy to trace: a re-render only happens when the developer says so.

### Trade-offs

- Forgetting to call `StateHasChanged()` is a common mistake for new users.
- Batch updates must be managed manually (call once after all mutations, not after each one).

---

## 6. Component Type Names Must Be Exported (Uppercase First Letter)

### The decision

All component structs must be exported Go types — their names must begin with an uppercase letter.

### Why

This is a consequence of decision #2. Since the compiler derives the Go type name from the PascalCase filename, and all component types must be resolvable via `go/packages` inspection, unexported (lowercase) types are not accessible across package boundaries and cannot be reliably referenced in generated code.

Additionally, component tags in templates are identified by their uppercase first letter, which is how the compiler distinguishes user-defined components from standard HTML elements.

---

## 7. Single-Slot Content Projection (No Named Slots)

### The decision

Layout components support only a **single content slot**, identified by a field of type `[]*vdom.VNode`. Named slots (multiple distinct projection points) are not currently supported.

### Why

Named slots require either a custom template syntax for slot assignment (e.g. `<template #header>`) or a more complex component protocol. Both options require deeper compiler and runtime changes that are deferred until the core model is proven stable.

A single slot covers the most common use case (a layout wrapping a page's body content) with minimal complexity.

### Trade-offs

- Components that need multiple distinct injection points (e.g. a card with separate header, body, and footer slots) cannot use content projection and must be composed differently.
- Named slots are planned for a future release once the single-slot model is validated.
