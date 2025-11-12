# Content Projection (Slots)

The framework supports content projection, allowing you to create reusable layout components that can render content provided by a parent. This is achieved through a type-driven system.

## How It Works

1. **Parent Provides Content:** A parent component provides content by placing it between the child component's opening and closing tags.

2. **Child Receives Content:** The child component defines an exported field of type `[]*vdom.VNode` to receive the content.

3. **Child Renders Content:** The child's template uses a standard data-binding expression `{...}` to render the VNode slice.

4. **Dev Warning:** When compiling with the `-dev-warnings` flag, a warning will be logged to the browser console if a component with a content slot is rendered with an empty or nil slice.

> **Important:** The compiler identifies the content projection slot by its **type**. While the convention is to name the field `Children` or `BodyContent`, any exported field of type `[]*vdom.VNode` will work. The type is what matters, not the name.

> **Constraint:** Components can have only **one** content slot. If a component has multiple `[]*vdom.VNode` fields, the compiler will error.

## Example: Creating a Reusable Card Component

### Child Go Code (`card.go`)

```go
package appcomponents

import "github.com/vcrobe/nojs/vdom"

// Card is a layout component with content projection.
type Card struct {
    Title       string
    // This field receives the projected content from the parent.
    // Any exported field of type []*vdom.VNode becomes a content slot.
    BodyContent []*vdom.VNode
}
```

### Child Template (`Card.gt.html`)

```html
<div class="card">
    <h2 class="card-title">{Title}</h2>
    <div class="card-body">
        {BodyContent}
    </div>
</div>
```

### Parent Template Usage (`app.gt.html`)

```html
<!-- Card with content -->
<Card Title="User Profile">
    <!-- This HTML is compiled into a []*vdom.VNode slice -->
    <!-- and passed to the 'BodyContent' field of the Card component. -->
    <p>This content is defined in the parent.</p>
    <button @onclick="SubmitProfile">Submit</button>
</Card>

<!-- Empty card (slot will be nil) -->
<Card Title="Empty Example"></Card>
```

## Generated Code

The compiler generates the following code for the examples above:

### Generated Card Component

```go
func (c *Card) Render(r *runtime.Renderer) *vdom.VNode {
    return vdom.Div(map[string]any{"class": "card"}, 
        vdom.NewVNode("h2", map[string]any{"class": "card-title"}, nil, c.Title),
        vdom.Div(map[string]any{"class": "card-body"}, 
            func() []*vdom.VNode {
                if len(c.BodyContent) == 0 {
                    console.Warn("[Slot] Rendering empty content slot 'BodyContent' in component 'Card'. Parent provided no content.")
                }
                return c.BodyContent
            }()...)) // Spread operator
}
```

### Generated Parent Component

```go
func (c *App) Render(r *runtime.Renderer) *vdom.VNode {
    return vdom.Div(nil,
        // Card with content
        r.RenderChild("Card_1", &Card{
            Title: "User Profile",
            BodyContent: []*vdom.VNode{
                vdom.Paragraph("This content is defined in the parent.", nil),
                vdom.Button("Submit", nil),
            },
        }),
        // Empty card
        r.RenderChild("Card_2", &Card{
            Title: "Empty Example",
            BodyContent: nil, // No content provided
        }),
    )
}
```

## Empty Slot Handling

When a parent provides no content, the slot field is set to `nil` (not an empty slice) to avoid unnecessary allocations:

```html
<Card Title="Empty"></Card>
```

Compiles to:

```go
&Card{Title: "Empty", BodyContent: nil}
```

Components can check if content was provided:

```go
func (c *Card) HasContent() bool {
    return len(c.BodyContent) > 0
}
```

## Advanced Patterns

### Conditional Rendering Based on Slot Content

```html
<!-- Card.gt.html -->
<div class="card">
    <h2>{Title}</h2>
    {@if HasContent}
        <div class="card-body">
            {BodyContent}
        </div>
    {@endif}
</div>
```

```go
// card.go
type Card struct {
    Title       string
    BodyContent []*vdom.VNode
    HasContent  bool
}

func (c *Card) UpdateHasContent() {
	c.HasContent = len(c.BodyContent) > 0
}
```

### Complex Slot Content

Slot content can include:
- HTML elements
- Data bindings
- Other components
- Loops (`{@for}`)
- Conditionals (`{@if}`)

```html
<Card Title="User List">
    <p>Total users: {UserCount}</p>
    {@for _, user := range Users trackBy user.ID}
        <UserProfile User={user}></UserProfile>
    {@endfor}
</Card>
```

## Type Safety

All slot content is validated at compile time:
- Component references must exist
- Data bindings must reference valid fields
- Event handlers must reference valid methods

Invalid slot content will produce compile errors:

```html
<Card Title="Invalid">
    <p>{NonExistentField}</p> <!-- Compile error -->
</Card>
```

## Component Composition vs. Typed Slots

For cases requiring type-safe, structured content, prefer **component composition** over slots:

### Using Slots (flexible, untyped content)

```go
type Card struct {
    Title   string
    Content []*vdom.VNode // Any content allowed
}
```

### Using Component Composition (typed, structured content)

```go
type DataTable struct {
    HeaderComponent *TableHeader // Typed structure
    RowComponent    *TableRow    // Typed structure
}

type TableHeader struct {
    Columns []string
}

type TableRow struct {
    Data User
}
```

Component composition provides:
- Type safety at compile time
- Clear contracts (component props as schema)
- Better IDE support and autocomplete
- Explicit parent-child relationships

Use **slots** for presentation flexibility. Use **component composition** for data contracts.

## Compilation Details

- Slot detection happens during component schema inspection
- Slot content collection happens during parent template compilation
- All validation occurs at compile time (zero runtime overhead)
- Empty slots compile to `nil` for efficiency
- Dev warnings are only included when `-dev-warnings` flag is used

## Limitations

- **Single slot per component**: Multiple `[]*vdom.VNode` fields will cause a compile error
- **Unnamed slots only**: No support for named slots (like `<slot name="header">`)
- **No default content**: Child components cannot define fallback content in templates (use computed properties instead)
