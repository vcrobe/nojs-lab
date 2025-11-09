# Data Binding Integration Tests

This package contains integration tests for the nojs framework's data binding feature.

## Overview

These tests verify the complete data binding pipeline:
1. **AOT Compilation**: Template expressions are compiled to Go code
2. **Component Rendering**: `Render()` generates VNode trees with bound data
3. **State Management**: Component state changes trigger re-renders
4. **VDOM Updates**: Re-renders produce updated VNode trees with new data

## Test Architecture

### In-Memory Testing (No Browser Required)

The tests run entirely in-memory without requiring a browser or WASM runtime:

- **TestRenderer**: Implements `runtime.Renderer` interface for test harness
- **Shared VNode**: Uses `vdom.VNode` directly (no test-specific duplication)
- **Shared Generated Code**: Tests use the SAME `Render()` method as production!

### File Structure

```
databinding/
├── Counter.gt.html              # Template with data binding expressions
├── counter.go                   # Component struct + business logic (NO build tags!)
├── Counter.generated.go         # AOT-generated Render() (NO build tags!)
├── counter_test.go              # Integration tests
└── README.md                    # This file
```

### Architecture Changes (Current Implementation)

**Key improvement:** All component code works in both WASM and test environments!

**Component Definition** (`counter.go`):
- **NO build tags!**
- Uses `runtime.ComponentBase` (no build tags as of latest refactor)
- Struct definition + business logic methods
- Works for both AOT compiler introspection AND test execution

**Generated Code** (`Counter.generated.go`):
- **NO build tags!**
- Uses `runtime.Renderer` interface (not concrete type)
- Uses `vdom.VNode` (available without build tags)
- **Same file compiles for WASM and tests!**

This represents a major simplification from earlier versions that required separate WASM and test harness files.

## Running Tests

```bash
# Run all data binding tests
cd testcomponents/databinding
go test -v

# Run specific test
go test -v -run TestDataBinding_StateUpdate
```

## Test Coverage

### TestDataBinding_InitialRender
Verifies that:
- Component renders with initial state values
- VDOM tree structure matches template
- Data binding expressions are correctly interpolated

### TestDataBinding_StateUpdate
Verifies that:
- State changes trigger re-renders via `StateHasChanged()`
- VDOM tree is updated with new values
- Multiple state changes produce correct results

### TestDataBinding_MultipleUpdates
Verifies that:
- Sequential state updates work correctly
- Each update produces expected VDOM
- State is properly preserved across renders

### TestDataBinding_RenderIsolation
Verifies that:
- Multiple component instances maintain separate state
- Renders don't affect other instances
- Test renderer correctly isolates components

## Adding New Tests

To add a new data binding test:

1. Create a new template file (`YourComponent.gt.html`)
2. Create component file (`yourcomponent.go` - NO build tags!)
   - Use `runtime.ComponentBase` for embedding
   - Add business logic methods
3. Run AOT compiler: `../../compiler/aotcompiler -in . -out .`
4. Write tests in `yourcomponent_test.go`

**That's it!** No need for separate test harness files. The component definition works in both WASM and test environments.

## Limitations

These tests verify:
✅ Data binding (struct fields → VNode content)
✅ State management (`StateHasChanged()` → re-render)
✅ VDOM tree generation
✅ **Generated Render() methods work identically in tests and production**

These tests do NOT verify:
❌ DOM rendering (`vdom.RenderToSelector`)
❌ VDOM diffing/patching algorithms
❌ Browser events and interactions
❌ Actual HTML output in browser

For end-to-end browser testing, use tools like Playwright or Selenium.

## Architecture Notes

**Key Achievement:** Complete architectural unification!

The framework now uses build-tag-free core types throughout:
- `runtime.ComponentBase` - NO build tags (latest refactor)
- `vdom.VNode` - NO build tags
- `runtime.Renderer` interface - NO build tags
- `runtime.Component` interface - NO build tags
- Generated `Render()` methods - NO build tags

This means:
✅ Single component file works everywhere
✅ Zero Render() duplication
✅ No separate test harness files needed
✅ Same code tested and deployed

Core types without build tags:
- `vdom/vnode_core.go` - VNode struct and constructors
- `runtime/component.go` - Component interface
- `runtime/renderer.go` - Renderer interface

See testcomponents/README.md for complete architecture details.
