# Test Components

This directory contains components specifically designed for testing the nojs framework, separate from demo/production components in `/appcomponents`.

## Purpose

Test components enable:
- **Integration testing** of framework features (data binding, events, lifecycle)
- **In-memory testing** without browser or WASM runtime
- **Full pipeline testing** from template compilation to VDOM generation

## Structure

```
testcomponents/
├── testrenderer.go           # Test harness for component rendering
├── databinding/              # Data binding integration tests
│   ├── Counter.gt.html
│   ├── counter.go            # Component (NO build tags!)
│   ├── Counter.generated.go  # AOT-generated (NO build tags!)
│   ├── counter_test.go       # Integration tests
│   └── README.md
└── README.md                 # This file
```

## Test Renderer

`testrenderer.go` provides:
- **TestRenderer**: Minimal renderer implementing `runtime.Renderer` interface
- Captures VDOM output for test assertions
- Uses shared `vdom.VNode` type (no duplication needed)

The test renderer implements `runtime.Renderer` interface:
- `RenderChild(key, child)` - Renders child components
- `ReRender()` - Triggered by `StateHasChanged()`
- `Navigate(path)` - No-op stub for tests

## Running Tests

```bash
# Run all test component tests
go test ./testcomponents/...

# Run specific package tests
go test ./testcomponents/databinding -v

# Run with coverage
go test ./testcomponents/... -cover
```

## Adding New Test Categories

To test a new feature (e.g., events, lifecycle):

1. Create a new directory: `testcomponents/featurename/`
2. Add template: `Component.gt.html`
3. Add WASM version: `component.go` (for AOT compiler)
4. Run compiler: `../../compiler/aotcompiler -in .`
5. Add test harness: `component_test_harness.go` (for go test)
6. Write tests: `component_test.go`
7. Document: `README.md`

## Build Tag Strategy

**Latest Architecture: Unified Build-Tag-Free Components**

As of the latest refactor, `runtime.ComponentBase` has NO build tags, which means component definitions no longer need separate WASM and test versions.

**All core types are now build-tag-free:**
- **`vdom/vnode_core.go`** - VNode type (no build tags)
- **`runtime/component.go`** - Component interface (no build tags)
- **`runtime/renderer.go`** - Renderer interface (no build tags)
- **`runtime/componentbase.go`** - ComponentBase struct (no build tags)

**Generated code** (e.g., `Counter.generated.go`) has NO build tags and compiles in both environments.

**Component definitions** (e.g., `counter.go`) have NO build tags and work everywhere.

**Only WASM runtime implementations use tags:**
- **`runtime/renderer_impl.go`** (`//go:build js || wasm`): RendererImpl with DOM operations
- **`vdom/render.go`** (`//go:build js || wasm`): DOM rendering via syscall/js

This represents a major simplification: **one component file works in both WASM and test environments!**

## What These Tests Cover

✅ Template compilation (AOT)  
✅ Data binding (struct → VNode content)  
✅ Component lifecycle (StateHasChanged)  
✅ VDOM tree generation  
✅ Component state management  
✅ **Shared generated Render() methods** (no duplication!)  

## What These Tests Don't Cover

❌ DOM rendering (requires browser)  
❌ VDOM diffing/patching  
❌ Browser events (click, input, etc.)  
❌ Actual HTML output  
❌ CSS rendering  

## Future Improvements

- [ ] Test VDOM diffing algorithm in isolation
- [ ] Mock browser events for event handler testing
- [ ] Test component lifecycle methods (OnInit, OnDestroy, etc.)
- [ ] Test content projection (slots)
- [ ] Test conditional rendering ({@if})
- [ ] Test list rendering ({@for})
- [ ] Auto-generate test harness boilerplate (struct + methods)
