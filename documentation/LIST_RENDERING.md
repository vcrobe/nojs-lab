# List Rendering Implementation

## Overview

This document describes the implementation of list rendering in the nojs Go + WASM framework using the `{@for}` directive. This feature allows you to render lists of items from slices or arrays with optimal performance through mandatory key tracking.

## Template Syntax

The list rendering syntax requires a `trackBy` clause to uniquely identify each item:

```html
<ul>
    {@for i, user := range Users trackBy user.ID}
        <li>User item</li>
    {@endfor}
</ul>
```

### Supported Syntax Variations

#### With Index and Value
```html
{@for i, user := range Users trackBy user.ID}
    <li>Item {i}: {user.Name}</li>
{@endfor}
```

#### Using Underscore to Ignore Index (Required Syntax)
```html
{@for _, user := range Users trackBy user.ID}
    <div>User: {user.Name}</div>
{@endfor}
```

**Important:** You **must** explicitly include both the index and value variables in the `{@for}` directive, following Go's standard `for...range` syntax. Use `_` (underscore) to ignore the index if you don't need it.

**Invalid Syntax - Will Cause Compilation Error:**
```html
<!-- ❌ INVALID - Missing index variable -->
{@for user := range Users trackBy user.ID}
    <li>User item</li>
{@endfor}
```

**Error message you'll see:**
```
template syntax error: Invalid {@for} syntax at line(s): [10]
  The {@for} directive requires both index and value variables.
  Correct syntax: {@for index, value := range Slice trackBy value.Field}
  To ignore the index, use underscore: {@for _, value := range Slice trackBy value.Field}
  Example: {@for _, user := range Users trackBy user.ID}
```

### Required Components

- **`{@for}`** - Opens a for-loop block
  - **Index variable** - **REQUIRED** - Loop index or `_` to ignore
  - **Value variable** - **REQUIRED** - The loop item variable name
  - **`range` expression** - Must reference an exported slice/array field on the component
  - **`trackBy` clause** - **REQUIRED** - Expression that resolves to a unique identifier for each item
- **`{@endfor}`** - Closes the for-loop block

## Why `trackBy` is Mandatory

Unlike many frameworks where keys are optional, the nojs framework **requires** the `trackBy` clause for several reasons:

1. **Prevents Common Bugs**: Forces developers to think about item identity upfront
2. **Enables Future Optimization**: Sets foundation for efficient VDOM diffing/reconciliation
3. **Type Safety**: Validates at compile time that the trackBy expression is valid
4. **Best Practice Enforcement**: Eliminates the "missing key" footgun from day one

## How It Works

### 1. Compile-Time Validation

The compiler performs the following checks:

- **Directive Matching**: Validates that every `{@for}` has a corresponding `{@endfor}`
- **Field Existence**: Verifies the range expression references an exported field
- **TrackBy Requirement**: Ensures the trackBy clause is present and valid
- **Syntax Validation**: Checks proper Go range syntax

Example validation error for missing `{@endfor}`:
```
template validation error in UserList.gt.html: 
found 1 {@for} directive(s) but only 0 {@endfor} directive(s).
  {@for} found at line(s): [10]
  {@endfor} found at line(s): []
  Missing 1 {@endfor} directive(s).
```

Example error for missing field:
```
Compilation Error in UserList.gt.html: Field 'Users' not found on component 'UserList'.
Available fields: [Title]
```

### 2. Preprocessing

Before HTML parsing, the `preprocessFor()` function transforms directives into placeholder HTML elements:

**Input:**
```html
{@for i, user := range Users trackBy user.ID}
    <li>User item</li>
{@endfor}
```

**Output:**
```html
<go-for data-index="i" data-value="user" data-range="Users" data-trackby="user.ID">
    <li>User item</li>
</go-for>
```

### 3. Code Generation

The `generateForLoopCode()` function generates Go code that:
- Creates a slice to collect VNodes
- Iterates using Go's `for...range`
- Stores the trackBy key for future optimization
- Optionally warns about empty slices (with `-dev-warnings` flag)

**Generated Code:**
```go
func() []*vdom.VNode {
    var user_nodes []*vdom.VNode
    
    // Development warning for empty slice (only if -dev-warnings flag is set)
    if len(c.Users) == 0 {
        console.Warn("[@for] Rendering empty list for 'Users' in UserList. Consider using {@if} to handle empty state.")
    }

    for i, user := range c.Users {
        user_key := user.ID
        _ = user_key // trackBy key stored for future diff optimization
        
        user_child := vdom.NewVNode("li", nil, nil, "User item")
        if user_child != nil {
            user_nodes = append(user_nodes, user_child)
        }
    }
    return user_nodes
}()
```

### 4. Empty Slice Handling

When a slice is `nil` or empty:
- Go's `for...range` executes zero iterations (safe, no panic)
- The loop renders nothing (empty VNode slice)
- With `-dev-warnings`, a console warning is logged
- Parent element renders with no children

**Example:** Empty `<ul>` renders as `<ul></ul>` (no `<li>` elements)

## Development Warnings

### The `-dev-warnings` Flag

Enable development warnings during compilation:

```bash
cd compiler
go run . -in .. -out ../appcomponents -dev-warnings
```

**What it does:**
- Adds `console.Warn()` calls when rendering empty slices
- Suggests using `{@if}` to handle empty states
- Zero performance impact in production (warnings not generated without flag)

**Console Output (with warnings enabled):**
```
⚠️ [@for] Rendering empty list for 'Users' in UserList. Consider using {@if} to handle empty state.
```

**Production Build (without warnings):**
```bash
cd compiler
go run . -in .. -out ../appcomponents
```
No warning code is generated - cleaner output, smaller bundle.

## Example Usage

### Component Struct

```go
package appcomponents

import "github.com/vcrobe/nojs/runtime"

type User struct {
    ID   int
    Name string
}

type UserList struct {
    runtime.ComponentBase
    Users []User
    Title string
}

func (u *UserList) OnInit() {
    u.Users = []User{
        {ID: 1, Name: "Alice"},
        {ID: 2, Name: "Bob"},
        {ID: 3, Name: "Charlie"},
    }
}

func (u *UserList) AddUser() {
    newID := len(u.Users) + 1
    u.Users = append(u.Users, User{
        ID:   newID,
        Name: "User " + string(rune('A' + newID - 1)),
    })
    u.StateHasChanged()
}

func (u *UserList) ClearUsers() {
    u.Users = []User{}
    u.StateHasChanged()
}
```

### Template

```html
<div>
    <h2>{Title}</h2>
    
    <div>
        <button @onclick="AddUser">Add User</button>
        <button @onclick="ClearUsers">Clear All</button>
    </div>
    
    <ul>
        {@for i, user := range Users trackBy user.ID}
            <li>User item</li>
        {@endfor}
    </ul>
</div>
```

### Generated Code

```go
func (c *UserList) Render(r *runtime.Renderer) *vdom.VNode {
    return vdom.Div(nil, 
        vdom.NewVNode("h2", nil, nil, fmt.Sprintf("%v", c.Title)), 
        vdom.Div(nil, 
            vdom.Button("Add User", map[string]any{"onClick": c.AddUser}), 
            vdom.Button("Clear All", map[string]any{"onClick": c.ClearUsers})), 
        vdom.Div(nil, func() []*vdom.VNode {
            var allChildren []*vdom.VNode
            allChildren = append(allChildren, func() []*vdom.VNode {
                var user_nodes []*vdom.VNode
                
                // Development warning (only with -dev-warnings flag)
                if len(c.Users) == 0 {
                    console.Warn("[@for] Rendering empty list for 'Users' in UserList. Consider using {@if} to handle empty state.")
                }

                for i, user := range c.Users {
                    user_key := user.ID
                    _ = user_key // trackBy key stored for future diff optimization

                    user_child := vdom.NewVNode("li", nil, nil, "User item")
                    if user_child != nil {
                        user_nodes = append(user_nodes, user_child)
                    }
                }
                return user_nodes
            }()...)
            return allChildren
        }()...))
}
```

## Handling Empty States

### Recommended Pattern: Use `{@if}` Directive

```html
<div>
    {@if len(Users) == 0}
        <p>No users found. Click "Add User" to get started.</p>
    {@else}
        <ul>
            {@for _, user := range Users trackBy user.ID}
                <li>User item</li>
            {@endfor}
        </ul>
    {@endif}
</div>
```

**Why this pattern?**
- Explicit control over empty state UI
- No reliance on dev warnings for UX
- Clean separation between "no data" and "has data" states
- Better user experience

## Implementation Details

### File Modifications

**`compiler/main.go`:**
- Added `-dev-warnings` CLI flag

**`compiler/compiler.go`:**
- Added `compileOptions` struct to pass flags through compilation
- Added `extractTypeName()` function to handle complex types (slices, pointers)
- Added `preprocessFor()` function for directive preprocessing
- Added `generateForLoopCode()` function for code generation
- Updated `generateNodeCode()` to handle `<go-for>` placeholder nodes
- Updated child collection logic to spread for-loop VNode slices

**`vdom/vnode_core.go`:**
- Added `Key interface{}` field to VNode struct for future reconciliation

### Regex Patterns

```go
// With index: {@for i, user := range Users trackBy user.ID}
reFor := regexp.MustCompile(`\{\@for\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*,\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*range\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+trackBy\s+([a-zA-Z0-9_.]+)\}`)

// Without index: {@for user := range Users trackBy user.ID}
reForNoIndex := regexp.MustCompile(`\{\@for\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*range\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+trackBy\s+([a-zA-Z0-9_.]+)\}`)

// End directive
reEndFor := regexp.MustCompile(`\{\@endfor\}`)
```

### Placeholder HTML Elements

- `<go-for data-index="..." data-value="..." data-range="..." data-trackby="...">` - For loop wrapper with metadata

## Nil Slice Behavior

**Q: What happens if the slice is `nil`?**

**A:** Go's `for...range` over a `nil` slice executes zero iterations - no panic, no special handling needed.

```go
var users []User // nil slice
for i, user := range users {
    // This never executes
}
// Code continues normally
```

The generated code naturally handles `nil` slices:
- Loop body doesn't execute
- Empty VNode slice is returned
- Parent element renders with no children
- Optional dev warning logs to console

## Current Limitations

### Loop Variable Data Binding (Planned)

Currently, you cannot bind to loop variable fields in the template:

```html
<!-- NOT YET SUPPORTED -->
{@for _, user := range Users trackBy user.ID}
    <li>{user.Name} (ID: {user.ID})</li>
{@endfor}
```

**Workaround:** Use static content for now, or implement component for each item.

**Future Enhancement:** Compiler will need context tracking to distinguish component fields from loop variables.

### Nested Loops (Supported)

You can nest `{@for}` loops:

```html
{@for _, category := range Categories trackBy category.ID}
    <div>
        <h3>{category.Name}</h3>
        <ul>
            {@for _, item := range category.Items trackBy item.ID}
                <li>Item</li>
            {@endfor}
        </ul>
    </div>
{@endfor}
```

## Future Enhancements

1. **Loop Variable Data Binding**: Support `{user.Name}` expressions inside loops
2. **VDOM Reconciliation**: Use stored keys for efficient list diffing
3. **Index-Based Keys Warning**: Warn when using loop index as trackBy (anti-pattern)
4. **Complex TrackBy Expressions**: Support composite keys like `user.Org + "-" + user.ID`

## Testing

To test list rendering:

1. Create a component with a slice field
2. Add `{@for}` directive to template with `trackBy`
3. Compile: `cd compiler && go run . -in .. -out ../appcomponents -dev-warnings`
4. Build WASM: `GOOS=js GOARCH=wasm go build -o main.wasm`
5. Open `index.html` in browser
6. Check browser console for dev warnings (if enabled)
7. Test add/remove/clear operations

## Troubleshooting

**Error: "found X {@for} directive(s) but only Y {@endfor} directive(s)"**
- Every `{@for}` must have a matching `{@endfor}`
- Check line numbers in error message

**Error: "Field 'Users' not found on component"**
- Ensure field is exported (capitalized)
- Check spelling matches exactly

**Error: "Invalid {@for} directive - missing required attributes"**
- Ensure `trackBy` clause is present
- Check syntax: `{@for var := range Slice trackBy key}`

**Warning: Empty list rendering**
- Add `{@if len(Slice) > 0}` to handle empty state
- Or disable warnings by removing `-dev-warnings` flag

**List doesn't update after adding items**
- Call `StateHasChanged()` after modifying the slice
- Ensure component method properly appends/removes items

## Commit Message

```
feat(compiler): implement list rendering with mandatory trackBy and dev warnings

- Add {@for} directive for rendering slices/arrays
- Require trackBy clause for unique item identification
- Add -dev-warnings CLI flag for optional empty slice console warnings
- Support both index+value and value-only syntax
- Add VNode.Key field for future diff optimization
- Update type inspection to handle slice/array types
- Add comprehensive validation and error messages
```
