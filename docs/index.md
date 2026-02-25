# nojs Documentation

nojs is a type-safe web framework for building browser applications in pure Go using WebAssembly.

## Why nojs

- Compile-time template validation for props, methods, and expressions
- Virtual DOM rendering with efficient patching
- Component model with idiomatic Go structs
- AOT compiler that transforms `*.gt.html` into generated `Render()` methods

## Project Status

nojs is currently MVP/experimental. APIs can change while the framework evolves.

## Start Here

1. Follow the [Getting Started](getting-started.md) guide.
2. Read the [Quick Guide](guides/quick-guide.md) for practical feature usage.
3. Explore architecture docs for runtime and router internals.

## Repository

- GitHub: <https://github.com/ForgeLogic/nojs>
- Demo app path: `app/`
- Framework packages: `nojs/`, `compiler/`, `router/`
