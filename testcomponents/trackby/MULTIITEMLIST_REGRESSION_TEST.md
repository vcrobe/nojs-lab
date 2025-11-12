# MultiItemList Test Component

## Overview

This test component (`MultiItemList`) is specifically designed to catch the compiler bug that was fixed where multiple child elements per loop iteration caused variable shadowing errors.

## Bug Context

**Original Issue**: When compiling `@for` loops with multiple sibling child elements, the compiler generated the same variable name for each child, causing Go compilation errors:

```go
// BROKEN - before fix
for _, item := range c.Items {
    item_child := r.RenderChild(...)  // First element
    if item_child != nil {
        item_nodes = append(item_nodes, item_child)
    }
    item_child := vdom.NewVNode(...)  // ← Error: item_child already declared!
    if item_child != nil {
        item_nodes = append(item_nodes, item_child)
    }
}
```

**Why Tests Passed Before**: The original `TagList` and `ProductList` test components only had a single `<li>` element per loop iteration, so they never triggered the variable shadowing bug.

## Component Design

`MultiItemList` renders **2 sibling `<li>` elements per loop iteration**:
- First `<li>`: Displays the item name
- Second `<li>`: Displays metadata (ID and loop index)

```html
{@for i, item := range Items trackBy item.ID}
    <li><strong>{item.Name}</strong></li>
    <li class="metadata">ID: {item.ID} | Index: {i}</li>
{@endfor}
```

## Generated Code (Fixed)

```go
// FIXED - after compiler fix
for _, item := range c.Items {
    item_child_0 := vdom.NewVNode(...)  // Unique name
    if item_child_0 != nil {
        item_nodes = append(item_nodes, item_child_0)
    }
    item_child_1 := vdom.NewVNode(...)  // Unique name
    if item_child_1 != nil {
        item_nodes = append(item_nodes, item_child_1)
    }
}
```

## Test Coverage

The `multiitemlist_test.go` file contains 5 comprehensive tests:

1. **InitialRender** - Verifies correct rendering of 3 items with 2 elements each = 6 total `<li>` elements
2. **AddItem** - Tests adding items and ensuring elements render correctly (4 items × 2 = 8 `<li>`)
3. **ClearItems** - Tests clearing all items removes all rendered children
4. **RenderIsolation** - Verifies multiple component instances maintain separate state
5. **VariableNamesUnique** - Regression test that validates compilation succeeds with multiple renders

## How It Prevents Regressions

If the compiler is accidentally reverted to generate duplicate variable names, any of these tests would fail at compile time with a "variable already declared" error, immediately catching the regression.

## Test Results

✅ All 5 tests passing
✅ All 18 trackBy tests passing (combined with TagList and ProductList)
✅ Validates compiler fix for multiple children per loop iteration
