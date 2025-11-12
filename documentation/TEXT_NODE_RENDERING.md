# Text Node Rendering

This document explains how the nojs framework handles text content within components and HTML elements using dedicated text nodes.

## Overview

Text nodes are a fundamental part of the DOM that represent pure text content without any HTML wrapper elements. The framework uses a special VNode type with `Tag: "#text"` to create clean text nodes via `document.createTextNode()`, avoiding unnecessary `<span>` wrappers that would pollute the DOM.

## Why Dedicated Text Nodes?

### The Problem

Initially, text content was handled in two ways:

1. **Simple text**: Stored in the `Content` field of VNodes and set via `element.textContent`
2. **Mixed content**: Elements with both text and child elements required complex workarounds

This approach had limitations:
- Mixing text and children was ambiguous
- Content projection (slots) couldn't contain plain text
- Setting `textContent` destroys all child nodes
- No clean way to represent text-only content in the VDOM tree

### The Solution

Introduce a first-class text node representation:

```go
func Text(content string) *VNode {
    return &VNode{
        Tag:     "#text",
        Content: content,
    }
}
```

**Benefits:**
✅ Text is a regular child VNode, not a special case  
✅ Works seamlessly in content projection (slots)  
✅ Clean DOM output (no wrapper elements)  
✅ Consistent with standard DOM API  
✅ Simplifies compiler-generated code

## Implementation

### 1. VNode Constructor

The `vdom` package provides a helper function to create text nodes:

```go
// vdom/vnode.go
func Text(content string) *VNode {
    return &VNode{
        Tag:     "#text",
        Content: content,
    }
}
```

**Usage in generated code:**
```go
vdom.NewVNode("a", 
    map[string]any{"href": "/about"},
    []*vdom.VNode{
        vdom.Text("Go to About Page"),  // Text as child
    },
    ""  // Empty Content field
)
```

### 2. Rendering Text Nodes

The renderer handles the `#text` tag specially:

```go
// vdom/render.go
func createElement(n *VNode) js.Value {
    switch n.Tag {
    case "#text":
        if n.Content == "" {
            return js.Undefined()
        }
        return doc.Call("createTextNode", n.Content)
    
    case "a", "nav", "span", /* ... */:
        el := doc.Call("createElement", n.Tag)
        
        // ... set attributes ...
        
        // Render children (including text nodes)
        if n.Children != nil {
            for _, child := range n.Children {
                childEl := createElement(child)  // Recursively create
                if childEl.Truthy() {
                    el.Call("appendChild", childEl)
                }
            }
        }
        
        return el
    }
}
```

**Key points:**
- `createTextNode()` creates a pure DOM text node
- Text nodes are appended like any other child element
- Empty text nodes return `js.Undefined()` and are skipped

### 3. Compiler Code Generation

The AOT compiler generates `vdom.Text()` calls for text content:

```go
// compiler/compiler.go
func (c *Compiler) generateNodeCode(n *html.Node, indentLevel int) string {
    switch n.Type {
    case html.TextNode:
        text := strings.TrimSpace(n.Data)
        if text == "" {
            return ""
        }
        
        // Generate expression for the text (handles data binding, ternaries, etc.)
        textExpr := c.generateTextExpression(text)
        
        // Wrap in vdom.Text() call
        return fmt.Sprintf("vdom.Text(%s)", textExpr)
    
    case html.ElementNode:
        // ... generate element code ...
    }
}
```

**Generated code examples:**

Static text:
```go
vdom.Text("Hello, World!")
```

Data binding:
```go
vdom.Text(c.Username)
```

Ternary expression:
```go
vdom.Text(func() string { if c.IsLoggedIn { return "Logout" } else { return "Login" } }())
```

### 4. Content Projection (Slots)

Text nodes work seamlessly in content projection:

**Template:**
```html
<Card>
    <h2>Card Title</h2>
    <p>Some description text</p>
    Click here for more!
</Card>
```

**Generated code:**
```go
r.RenderChild("Card_1", &appcomponents.Card{
    Children: []*vdom.VNode{
        vdom.NewVNode("h2", nil, nil, "Card Title"),
        vdom.NewVNode("p", nil, nil, "Some description text"),
        vdom.Text("Click here for more!"),  // Text as slot content
    },
})
```

**Card component:**
```go
type Card struct {
    runtime.ComponentBase
    Children []*vdom.VNode  // Slot for projected content
}

func (c *Card) Render(r runtime.Renderer) *vdom.VNode {
    return vdom.NewVNode("div", 
        map[string]any{"class": "card"},
        c.Children,  // Children includes text nodes
        "",
    )
}
```

**Rendered DOM:**
```html
<div class="card">
    <h2>Card Title</h2>
    <p>Some description text</p>
    Click here for more!
</div>
```

## Edge Cases and Gotchas

### 1. Empty Text Nodes

Empty or whitespace-only text nodes are skipped during generation:

```go
text := strings.TrimSpace(n.Data)
if text == "" {
    return ""  // Don't generate code for empty text
}
```

This prevents unnecessary text nodes in the DOM.

### 2. Text Content vs. Children

VNodes can have **either** `Content` (string) **or** `Children` (slice of VNodes), but not both meaningfully:

```go
// ✅ Correct: Text-only element using Content
vdom.NewVNode("p", nil, nil, "Hello")

// ✅ Correct: Element with children (including text nodes)
vdom.NewVNode("p", nil, []*vdom.VNode{
    vdom.Text("Hello "),
    vdom.NewVNode("strong", nil, nil, "World"),
}, "")

// ❌ Ambiguous: What takes precedence?
vdom.NewVNode("p", nil, []*vdom.VNode{
    vdom.Text("Child text"),
}, "Content text")
```

**Rule**: If `Children` is non-empty, `Content` should be empty string.

### 3. Patching Text Nodes

During diff/patch, the `Content` field must only be updated when there are no children:

```go
// vdom/render.go (patchElement)
if len(newVNode.Children) == 0 && oldVNode.Content != newVNode.Content {
    domElement.Set("textContent", newVNode.Content)
}
```

Setting `textContent` destroys all child nodes, so we skip it when children exist. This was a critical bug that caused text nodes to disappear during patching.

### 4. Event Listeners on Text Nodes

Text nodes **cannot** have event listeners:

```html
<!-- ❌ Invalid: Text nodes don't support events -->
<div>
    Some text that should be clickable
</div>
```

Wrap text in an element if you need event handling:

```html
<!-- ✅ Correct: Wrap in clickable element -->
<div>
    <span @onclick="HandleClick">Some text that should be clickable</span>
</div>
```

## HTML Tag Conflicts

Component names that match HTML tags can cause parsing issues due to case-insensitivity:

### The Problem

```html
<!-- Component name: Link -->
<Link Href="/about">Go to About</Link>
```

The HTML parser treats `<Link>` as `<link>` (case-insensitive), which is a void element that cannot have children. The text "Go to About" is lost.

### The Solution

The compiler validates component names against a list of problematic HTML tags:

```go
// compiler/compiler.go
var problematicHTMLTags = map[string]string{
    "link":     "RouterLink, NavLink, or HyperLink",
    "form":     "DataForm or FormComponent",
    "button":   "ButtonComponent or ActionButton",
    "input":    "InputComponent or TextField",
    "select":   "SelectComponent or Dropdown",
    "textarea": "TextArea or MultilineInput",
    "image":    "ImageComponent or Picture",
    "meta":     "MetaData or MetaComponent",
    "style":    "StyleComponent or StyledElement",
    "script":   "ScriptComponent or ScriptTag",
}

func (c *Compiler) validateComponentName(name string) error {
    lowerName := strings.ToLower(name)
    if suggestion, exists := problematicHTMLTags[lowerName]; exists {
        return fmt.Errorf(
            "component name '%s' conflicts with HTML tag '%s'. Consider renaming to: %s",
            name, lowerName, suggestion,
        )
    }
    return nil
}
```

**Example error:**
```
component name 'Link' conflicts with HTML tag 'link'. Consider renaming to: RouterLink, NavLink, or HyperLink
```

### Workaround

Use `<a>` tags directly with event handlers:

```html
<a href="/about" @onclick="NavigateToAbout">Go to About</a>
```

Or rename the component:

```html
<RouterLink Href="/about">Go to About</RouterLink>
```

## Performance Considerations

### Text Node Creation

Creating text nodes is very efficient:

```javascript
document.createTextNode("Hello")  // Fast native browser operation
```

Text nodes are lighter than element nodes:
- No attributes to process
- No event listeners to attach
- No tag name lookup

### Memory Footprint

Text nodes in the VDOM have minimal overhead:

```go
&VNode{
    Tag:     "#text",      // 6 bytes + string header
    Content: "some text",  // String length + header
    // Other fields are zero-valued
}
```

Element nodes require more memory:
- Attributes map
- Children slice
- Event listener functions

## Testing Text Nodes

**Manual testing:**
1. Build and serve the application
2. Inspect the DOM in browser DevTools
3. Verify text appears as text nodes, not wrapped in elements
4. Check that text updates correctly during re-renders

**Example inspection:**
```html
<a href="/about">
  #text "Go to About Page"  <!-- Text node, not <span> -->
</a>
```

## Future Enhancements

Potential improvements:

1. **Text Node Pooling**: Reuse text node objects to reduce allocations
2. **Formatted Text**: Support for `\n`, `\t`, and other whitespace handling
3. **Text Measurement**: Helper functions to measure text width/height for layout
4. **Rich Text**: Integration with contenteditable for in-place editing
5. **Text Diffing**: Character-level diffing for more efficient text updates

## Conclusion

Text nodes are a core primitive that enables clean, efficient text rendering throughout the framework. By treating text as first-class VNode children rather than a special case, we achieve:

- **Consistency**: Text works the same everywhere (slots, elements, components)
- **Simplicity**: No special handling in component code
- **Performance**: Direct use of browser's native text node API
- **Correctness**: Proper interaction with patching algorithm

The `vdom.Text()` helper and `#text` tag convention provide a clean abstraction that maps directly to DOM concepts while remaining easy to understand and use.
