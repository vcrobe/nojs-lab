# Inline Conditional Expressions

This document describes the **Inline Conditional Expressions** feature that allows you to write conditional logic directly inside component templates to control attribute values and text content dynamically.

## Overview

The compiler validates all conditions at compile time, ensuring they are exported boolean fields on your component's struct. This provides type safety and catches errors before runtime.

## Feature Patterns

### 1. Conditional Attributes and Text

Use ternary-like expressions with the syntax `{ condition ? value_if_true : value_if_false }` to control attribute values or text content.

**Requirements:**
- Single quotes (`'`) only for string values
- Condition must be an exported boolean field
- Strict boolean type enforcement (no truthy/falsy evaluation)

#### Example: Conditional CSS Class

```html
<!-- Adds the 'error' class only if HasError is true -->
<div class="message {HasError ? 'error' : 'success'}">
    Status message
</div>
```

**Generated Go code:**
```go
map[string]any{"class": fmt.Sprintf("message %s", func() string {
    if c.HasError {
        return "error"
    }
    return "success"
}())}
```

#### Example: Conditional Text Content

```html
<!-- Changes the button text based on the IsSaving state -->
<button>
    { IsSaving ? 'Saving...' : 'Save Changes' }
</button>
```

**Generated Go code:**
```go
func() string {
    if c.IsSaving {
        return "Saving..."
    }
    return "Save Changes"
}()
```

### 2. Boolean Attribute Shorthand

For standard HTML boolean attributes (like `disabled`, `checked`, `readonly`), use a simple boolean expression. The framework will add or remove the attribute accordingly.

**Requirements:**
- Only works with predefined standard boolean attributes
- Condition must be an exported boolean field
- Use full ternary syntax for non-standard boolean attributes (e.g., `aria-disabled`)

#### Example: Disabling a Button

```html
<!-- The 'disabled' attribute is added only if IsSaving is true -->
<button disabled="{IsSaving}">Submit</button>
```

**Generated Go code:**
```go
map[string]any{"disabled": c.IsSaving}
```

#### Standard Boolean Attributes List

The compiler recognizes these standard HTML boolean attributes:
- `disabled`, `checked`, `readonly`, `required`
- `autofocus`, `autoplay`, `controls`, `loop`, `muted`
- `selected`, `hidden`, `multiple`, `novalidate`
- `open`, `reversed`, `scoped`, `seamless`, `sortable`
- `truespeed`, `default`, `ismap`, `formnovalidate`

### 3. Negation Operator

Use the `!` operator to negate boolean conditions in both ternary expressions and boolean shorthand.

#### Example: Negation in Boolean Shorthand

```html
<!-- Button is disabled when IsReady is false -->
<button disabled="{!IsReady}">
    {IsReady ? 'Submit' : 'Please wait...'}
</button>
```

**Generated Go code:**
```go
map[string]any{"disabled": !c.IsReady}
```

#### Example: Negation in Ternary Expression

```html
<p>{!IsComplete ? 'In Progress' : 'Completed'}</p>
```

**Generated Go code:**
```go
func() string {
    if !c.IsComplete {
        return "In Progress"
    }
    return "Completed"
}()
```

### 4. Non-Standard Boolean Attributes

For attributes not in the standard boolean list (e.g., ARIA attributes), you must use the full ternary expression with explicit `'true'` and `'false'` string values.

#### Example: Explicit `aria-disabled`

```html
<button aria-disabled="{IsSaving ? 'true' : 'false'}">
    Submit
</button>
```

**Generated Go code:**
```go
map[string]any{"aria-disabled": func() string {
    if c.IsSaving {
        return "true"
    }
    return "false"
}()}
```

### 5. Multiple Conditionals

You can combine multiple conditional expressions in a single attribute value.

#### Example: Multiple Classes

```html
<div class="box {IsActive ? 'active' : 'inactive'} {IsLarge ? 'large' : 'small'}">
    Content here
</div>
```

**Generated Go code:**
```go
map[string]any{"class": fmt.Sprintf("box %s %s", func() string {
    if c.IsActive {
        return "active"
    }
    return "inactive"
}(), func() string {
    if c.IsLarge {
        return "large"
    }
    return "small"
}())}
```

## Compile-Time Validation

The compiler performs strict validation to ensure type safety:

### 1. Field Existence Check

```html
<!-- ERROR: InvalidField doesn't exist on component -->
<button disabled="{InvalidField}">Test</button>
```

**Error message:**
```
Compilation Error in Component.gt.html:2: Condition 'InvalidField' not found on component 'MyComponent'. 
Available fields: [IsReady, IsSaving, HasError]
```

### 2. Type Check

```html
<!-- ERROR: Count is an int, not a bool -->
<button disabled="{Count}">Test</button>
```

**Error message:**
```
Compilation Error in Component.gt.html:2: Condition 'Count' must be a bool field, found type 'int'.
```

### 3. Boolean Attribute Restriction

```html
<!-- ERROR: aria-disabled is not a standard boolean attribute -->
<button aria-disabled="{IsDisabled}">Test</button>
```

**Error message:**
```
Compilation Error in Component.gt.html:2: Boolean shorthand syntax can only be used with 
standard HTML boolean attributes. For attribute 'aria-disabled', use the full ternary 
expression: {IsDisabled ? 'true' : 'false'}
```

## Complete Example

### Component Go File

```go
//go:build js && wasm

package appcomponents

type FormDemo struct {
    IsSaving   bool
    IsValid    bool
    HasError   bool
    IsLocked   bool
}

func (f *FormDemo) Submit() {
    f.IsSaving = true
    // ... submission logic
}
```

### Component Template

```html
<div class="form {HasError ? 'form-error' : 'form-valid'}">
    <input 
        type="text" 
        readonly="{IsLocked}"
        class="{IsValid ? 'input-valid' : 'input-invalid'}" 
    />
    
    <button 
        disabled="{IsSaving}" 
        @onclick="Submit"
        aria-busy="{IsSaving ? 'true' : 'false'}">
        {IsSaving ? 'Submitting...' : 'Submit Form'}
    </button>
    
    <p>{HasError ? 'Please fix the errors' : 'Form is ready'}</p>
</div>
```

## Design Decisions

1. **No nested ternaries**: Keeps templates simple and readable
2. **No mixed data binding with conditionals**: Each expression type is separate
3. **Single quotes only**: Consistent syntax that's easy to parse
4. **Strict boolean types**: Prevents common bugs from truthy/falsy confusion
5. **Negation support**: Provides flexibility without complexity

## Performance

All conditional logic is compiled to efficient Go code with inline closures. The generated code has minimal runtime overhead and benefits from Go's compiler optimizations.
