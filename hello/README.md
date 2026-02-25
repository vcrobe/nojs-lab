# hello

`hello` is the **smallest possible demo app** for this framework.

Its goal is to help a new user quickly understand the core flow:

1. Write a component
2. Use a template (`*.gt.html`)
3. Compile templates into `Render()` code
4. Build to WebAssembly and run in the browser

## Why this module exists

This app must stay intentionally minimal so the essence of the framework is easy to grasp in a few minutes.

- No router usage
- No advanced app architecture
- No extra abstractions beyond what is required to render components

If something can be removed without breaking the core demo purpose, it should be removed.

## Scope of this demo

Included:
- Component rendering demonstration
- Template compilation demonstration
- WASM build + browser run

Excluded:
- Router and route registration
- Complex state flows
- Production-level app structure

## Run the hello demo

From repository root:

1. Build templates and WASM (dev flow):

	```bash
	make full
	```

2. Serve the project (example):

	```bash
	make serve
	```

3. Open:

	```
	http://localhost:9090/hello/wwwroot/
	```

## What to read in this module

- `hello/internal/app/main.go` — app entrypoint and component mount
- `hello/internal/app/components/` — minimal component examples
- `hello/wwwroot/index.html` — static host page

## Keep it simple policy

This module is a learning artifact first.

When editing it, prefer:
- fewer files
- shorter code paths
- explicit and readable examples

Avoid adding features here unless they are necessary to explain the framework basics.
