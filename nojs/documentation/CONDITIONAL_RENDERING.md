# Conditional Rendering Implementation

## Overview

This document describes the implementation of block-based conditional rendering in the nojs Go + WASM framework. This feature allows you to show, hide, or swap out entire chunks of UI based on boolean conditions in your component's state.

## Template Syntax

The conditional rendering syntax uses block directives similar to other template languages:

```html
{@if IsSaved}
    <p class="success">Your changes have been saved!</p>
{@else if IsLoading}
    <p class="loading">Saving...</p>
{@else}
    <button @onclick="SaveChanges">Save Changes</button>
{@endif}
```

### Supported Directives

- **`{@if Condition}`** - Opens a conditional block. `Condition` must be an exported boolean field on the component struct.
- **`{@else if Condition}`** - Optional else-if branch. You can have multiple else-if blocks.
- **`{@else}`** - Optional final else branch for when all conditions are false.
- **`{@endif}`** - Closes the conditional block.

## How It Works

### 1. Compile-Time Validation

The compiler performs the following checks at compile time:

- **Directive Matching**: Validates that every `{@if}` has a corresponding `{@endif}` directive
- **Field Existence**: Verifies that the condition field exists on the component struct
- **Type Safety**: Ensures the condition field is of type `bool`
- **Clear Error Messages**: Provides helpful compilation errors with line numbers and context if validation fails

Example validation error for missing `{@endif}`:
```
template validation error in /home/rob/projects/_lab/Golang/nojs/appcomponents/SimpleMessage.gt.html: 
found 1 {@if} directive(s) but only 0 {@endif} directive(s).
  {@if} found at line(s): [7]
  {@endif} found at line(s): []
  Missing 1 {@endif} directive(s).
```

Example error for invalid condition field:
```
Compilation Error in SimpleMessage.gt.html:8: Condition 'IsSaved' not found on component 'SimpleMessage'.
Available fields: [Counter, FirstProp, IsLoading]
```

### 2. Preprocessing

Before HTML parsing, the `preprocessConditionals()` function transforms template directives into placeholder HTML elements:

**Input:**
```html
{@if IsSaved}
    <p>Saved!</p>
{@else if IsLoading}
    <p>Loading...</p>
{@else}
    <button>Save</button>
{@endif}
```

**Output:**
```html
<go-conditional><go-if data-cond="IsSaved">
    <p>Saved!</p>
</go-if><go-elseif data-cond="IsLoading">
    <p>Loading...</p>
</go-if></go-elseif><go-else>
    <button>Save</button>
</go-if></go-elseif></go-else></go-conditional>
```

The `<go-conditional>` wrapper ensures all branches are properly nested as siblings within the HTML parser.

### 3. Code Generation

The `generateConditionalCode()` function generates an **Immediately Invoked Function Expression (IIFE)** that returns a VNode:

```go
func() *vdom.VNode {
    if c.IsSaved {
        return vdom.Paragraph("Saved!", map[string]any{"class": "success"})
    } else if c.IsLoading {
        return vdom.Paragraph("Loading...", map[string]any{"class": "loading"})
    } else {
        return vdom.Button("Save", map[string]any{"onClick": c.SaveChanges}, )
    }
}()
```

**Key Benefits:**
- **Lazy Evaluation**: Only the matching branch's VNodes are created
- **Performance**: No unnecessary DOM tree construction for hidden branches
- **Clean Code**: Standard Go if/else logic, easy to debug

### 4. Runtime Behavior

At runtime, when the component renders:
1. The condition is evaluated (e.g., `c.IsSaved`)
2. Only the matching branch executes its VNode creation code
3. The resulting VNode is returned and rendered to the DOM
4. When state changes trigger a re-render, the conditions are re-evaluated

## Example Usage

### Component Struct

```go
type SimpleMessage struct {
    runtime.ComponentBase
    Counter   int
    FirstProp string
    IsSaved   bool     // Must be exported and bool type
    IsLoading bool     // Must be exported and bool type
}
```

### Template

```html
<div>
    <p>Simple Message Component</p>
    <p>Counter: {Counter}</p>

    {@if IsSaved}
        <p class="success">Your changes have been saved!</p>
    {@else if IsLoading}
        <p class="loading">Saving...</p>
    {@else}
        <button @onclick="Increment">Click me</button>
    {@endif}
</div>
```

### Generated Code

```go
func (c *SimpleMessage) Render(r *runtime.Renderer) *vdom.VNode {
    return vdom.Div(nil, 
        vdom.Paragraph("Simple Message Component", nil), 
        vdom.Paragraph(fmt.Sprintf("Counter: %v", c.Counter), nil), 
        func() *vdom.VNode {
            if c.IsSaved {
                return vdom.Paragraph("Your changes have been saved!", 
                    map[string]any{"class": "success"})
            } else if c.IsLoading {
                return vdom.Paragraph("Saving...", 
                    map[string]any{"class": "loading"})
            } else {
                return vdom.Button("Click me", 
                    map[string]any{"onClick": c.Increment}, )
            }
        }())
}
```

## Implementation Details

### File Modifications

**`compiler/compiler.go`:**
- Added `preprocessConditionals()` function
- Added `generateConditionalCode()` function
- Updated `compileComponentTemplate()` to call preprocessor
- Updated `generateNodeCode()` to handle `go-conditional` nodes

### Regex Patterns

```go
reIf := regexp.MustCompile(`\{\@if ([^}]+)\}`)
reElseIf := regexp.MustCompile(`\{\@else if ([^}]+)\}`)
reElse := regexp.MustCompile(`\{\@else\}`)
reEndIf := regexp.MustCompile(`\{\@endif\}`)
```

### HTML Placeholder Elements

- `<go-conditional>` - Wrapper for the entire conditional structure
- `<go-if data-cond="...">` - If branch with condition attribute
- `<go-elseif data-cond="...">` - Else-if branch with condition attribute
- `<go-else>` - Else branch (no condition needed)

## Future Enhancements

Potential improvements for future versions:

1. **Nested Conditionals**: Support conditional blocks inside other conditionals
2. **Complex Conditions**: Support boolean expressions like `IsSaved && !IsLoading`
3. **Switch-Case**: Add `{@switch}` directive for multiple conditions on the same value
4. **Loops with Conditionals**: Combine `{@for}` loops with conditional rendering
5. **Performance Metrics**: Add compiler flags to measure VNode creation overhead

## Testing

To test conditional rendering:

1. Create a component with boolean fields
2. Add conditional blocks to the template
3. Compile: `cd compiler && go run . -in=..`
4. Build WASM: `GOOS=js GOARCH=wasm go build -o main.wasm`
5. Open `index.html` in a browser and toggle the boolean fields

## Troubleshooting

**Error: "found X {@if} directive(s) but only Y {@endif} directive(s)"**
- Every `{@if}` must have a matching `{@endif}`
- Check the line numbers in the error message to locate missing directives
- Common mistake: forgetting `{@endif}` at the end of conditional blocks

**Error: "found X {@endif} directive(s) but only Y {@if} directive(s)"**
- You have extra `{@endif}` directives without matching `{@if}`
- Check the line numbers to find and remove extra directives

**Error: "Condition 'X' not found on component"**
- Ensure the field is exported (capitalized)
- Check spelling matches exactly

**Error: "Condition 'X' must be a bool field"**
- Change the field type to `bool`

**Unexpected behavior: Wrong branch renders**
- Check the order of your `else if` conditions
- Verify boolean field values at runtime with `println()`

## Commit Message

```
feat(compiler): implement block-based conditional rendering with directive matching validation
```
