# Multi-line HTML Tag Tests

This test package verifies that the AOT compiler correctly handles both single-line and multi-line HTML tags with data bindings.

## Problem

Previously, the AOT compiler would fail when HTML tags with data bindings were written across multiple lines:

```html
<!-- ❌ This would fail -->
<h1>
    Blog Posts for the Year {Year}
</h1>
```

Error: `failed to format generated code: string literal not terminated`

## Solution

The compiler now:
1. **Concatenates all text nodes** within an element (not just the first child)
2. **Properly escapes newlines** in the generated Go code using `strconv.Quote()`

## Test Coverage

### TestMultilineTextSingleLine
Verifies that both single-line and multi-line formats produce correct VDOM output:

- ✅ Single-line: `<h1>Title</h1>` → Content: `"Title"`
- ✅ Multi-line: `<h1>\n    Title\n</h1>` → Content: `"\n    Title\n"` (whitespace preserved)

### TestMultilineTextDataBinding
Tests data binding with various edge cases:

- Empty strings
- Special characters (`<`, `>`, `&`, `"`, tabs)
- Very long text (1000+ characters)
- Negative numbers

### TestMultilineWhitespacePreservation
Specifically validates that:

- Multi-line tags preserve leading/trailing whitespace
- Single-line tags do NOT have extraneous newlines

## Affected HTML Tags

The fix applies to all text-containing tags:

- Headings: `<h1>`, `<h2>`, `<h3>`, `<h4>`, `<h5>`, `<h6>`
- Paragraphs: `<p>`
- Buttons: `<button>`
- List items: `<li>`
- Form elements: `<option>`
- Semantic elements: `<a>`, `<span>`, `<nav>`, `<section>`, etc.

## Usage

Both formats are now fully supported:

```html
<!-- Single-line (works) -->
<h1>Blog Posts for {Year}</h1>

<!-- Multi-line (also works!) -->
<h1>
    Blog Posts for {Year}
</h1>
```
