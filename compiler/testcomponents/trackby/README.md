# TrackBy Test Components

This directory tests both formats of the `trackBy` clause:

1. **Bare Variable** - For primitive types (string, int, etc.)
2. **Dot-Notation** - For struct fields (user.ID, etc.)

## Components

### TagList
Tests bare variable `trackBy` with a slice of primitive strings:
```go
type TagList struct {
    Tags []string  // Slice of primitive strings
}
```

Template:
```html
{@for _, tag := range Tags trackBy tag}
    <li>Tag: {tag}</li>
{@endfor}
```

### ProductList
Tests dot-notation `trackBy` with a slice of structs:
```go
type Product struct {
    ID   int
    Name string
}

type ProductList struct {
    Products []Product  // Slice of structs
}
```

Template:
```html
{@for _, product := range Products trackBy product.ID}
    <li>Product: {product.Name}</li>
{@endfor}
```

## Building

Both test components can be compiled together:

```bash
cd compiler
go run . -in ../testcomponents/trackby
```

Expected output:
```
🎉 Compilation completed successfully!
```

Both `trackBy` formats compile without errors and generate proper Render() methods.

## Key Differences

| Feature | Bare Variable | Dot-Notation |
|---------|---------------|--------------|
| Use Case | Primitive types (string, int, bool) | Struct fields |
| Syntax | `trackBy varName` | `trackBy varName.Field` |
| Example | `trackBy tag` | `trackBy product.ID` |
| Validation | Validates match with loop variable | Validates field exists on struct type |

