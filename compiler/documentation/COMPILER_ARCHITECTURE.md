# Compiler Architecture

This document describes the internal structure of the **nojs AOT compiler** (`github.com/ForgeLogic/nojs-compiler`). It explains how the package is organised, what each file is responsible for, and how data flows from a `.gt.html` template to a `.generated.go` file.

---

## Table of Contents

1. [Overview](#overview)
2. [File Summary](#file-summary)
3. [Key Types](#key-types)
4. [Compilation Pipeline](#compilation-pipeline)
5. [File Reference](#file-reference)
   - [compiler.go](#compilergo)
   - [types.go](#typesgo)
   - [preprocessor.go](#preprocessorgo)
   - [helpers.go](#helpersgo)
   - [validator.go](#validatorgo)
   - [discovery.go](#discoverygo)
   - [typeresolver.go](#typeresolvergo)
   - [codegen_attributes.go](#codegen_attributesgo)
   - [codegen_text.go](#codegen_textgo)
   - [codegen_loops.go](#codegen_loopsgo)
   - [codegen_conditionals.go](#codegen_conditionalsgo)
   - [codegen_nodes.go](#codegen_nodesgo)
   - [codegen.go](#codegengo)

---

## Overview

The nojs AOT compiler reads `.gt.html` template files, inspects the matching Go struct (props, state, methods), and generates a `.generated.go` file next to each template. The generated file contains two methods:

- **`Render(r runtime.Renderer) *vdom.VNode`** — builds the virtual DOM tree for the component.
- **`ApplyProps(source runtime.Component)`** — copies incoming props onto the component without touching internal state.

The compiler is invoked via the `nojsc` CLI binary (`cmd/nojsc/main.go`) or programmatically through the single public function `Compile(srcDir string, devMode bool) error`.

---

## File Summary

| File | Lines (approx.) | Responsibility |
|---|---|---|
| `compiler.go` | ~45 | Public API entry point — `Compile()` only |
| `types.go` | ~90 | All shared structs, package-level vars, and compiled regexes |
| `preprocessor.go` | ~130 | Source transformation: `{@for}` and `{@if}` rewriting before HTML parse |
| `helpers.go` | ~180 | Shared utilities: line estimation, DOM traversal, field/method name listing |
| `validator.go` | ~160 | Compile-time semantic validation and friendly error messages |
| `discovery.go` | ~230 | Filesystem scan + Go AST inspection to build `componentInfo` records |
| `typeresolver.go` | ~210 | Resolves dotted field paths (e.g. `Ctx.Title`) through Go AST |
| `codegen_attributes.go` | ~220 | Generates VNode attribute maps, ternary expressions, struct literals |
| `codegen_text.go` | ~180 | Text node data binding and slot child collection |
| `codegen_loops.go` | ~200 | `{@for}` loop VNode code generation |
| `codegen_conditionals.go` | ~180 | `{@if}/{@else if}/{@else}` VNode code generation |
| `codegen_nodes.go` | ~290 | Central dispatch: `generateNodeCode` routes each HTML node to the right generator |
| `codegen.go` | ~140 | Template pipeline: `compileComponentTemplate`, `generateApplyPropsBody` |

---

## Key Types

All types are declared in `types.go` and are package-private (lowercase in the package, exported only through `compiler.go`).

### `componentInfo`
Holds everything discovered about a single component before code generation:

```go
type componentInfo struct {
    Path          string          // Absolute path to the .gt.html template
    PascalName    string          // e.g. "CounterPage"
    LowercaseName string          // e.g. "counterpage" — used as map key
    PackageName   string          // Go package name (e.g. "pages")
    ImportPath    string          // Full import path (e.g. "github.com/ForgeLogic/nojs/app/internal/app/components/pages")
    Schema        componentSchema // Introspected props, state, methods, and slot
}
```

### `componentSchema`
Describes the fields and methods of the matching Go struct:

```go
type componentSchema struct {
    Props   map[string]propertyDescriptor // Input props (copied by ApplyProps)
    State   map[string]propertyDescriptor // Internal state (not copied)
    Methods map[string]methodDescriptor   // Event handlers and other methods
    Slot    *propertyDescriptor           // Optional []*vdom.VNode content slot
}
```

### `compileOptions`
Compiler flags threaded through `compileComponentTemplate` and all codegen functions:

```go
type compileOptions struct {
    DevMode          bool           // Enable runtime warnings (console.Warn calls in generated code)
    ComponentCounter map[string]int // Per-template counter, ensures unique RenderChild keys
}
```

### `loopContext`
Carries loop variable names into nested code generators so bindings like `{item.Name}` can be resolved:

```go
type loopContext struct {
    IndexVar string // e.g. "i"
    ValueVar string // e.g. "item"
}
```

---

## Compilation Pipeline

```
srcDir
  │
  ▼
discoverAndInspectComponents()          ← discovery.go
  │  Walk filesystem for *.gt.html
  │  Parse matching *.go file via go/ast
  │  Build []componentInfo
  │
  ▼
for each componentInfo:
  compileComponentTemplate()            ← codegen.go
    │
    ├─ os.ReadFile(.gt.html)
    │
    ├─ preprocessConditionals()         ← preprocessor.go
    │    Rewrites {@if}/{@else} blocks into <go-if>/<go-else> nodes
    │
    ├─ preprocessFor()                  ← preprocessor.go
    │    Rewrites {@for} blocks into <go-for> nodes
    │
    ├─ html.Parse()  (net/html)
    │    Produces a *html.Node tree
    │
    ├─ findBody() / findFirstElementChild()  ← helpers.go
    │    Navigates to the template root element
    │
    ├─ collectUsedComponents()          ← discovery.go
    │    Determines cross-package imports needed in generated file
    │
    ├─ generateNodeCode()               ← codegen_nodes.go
    │    Recursively walks the html.Node tree
    │    │
    │    ├─ TextNode        → generateTextExpression()    ← codegen_text.go
    │    ├─ <go-conditional>→ generateConditionalCode()   ← codegen_conditionals.go
    │    ├─ <go-for>        → generateForLoopCode()       ← codegen_loops.go
    │    ├─ ComponentTag    → generateStructLiteral()     ← codegen_attributes.go
    │    └─ HTMLElement     → generateAttributesMap()     ← codegen_attributes.go
    │
    ├─ generateApplyPropsBody()         ← codegen.go
    │    Produces prop-copy assignments for ApplyProps method
    │
    ├─ format.Source()  (go/format)
    │    Gofmt-formats the generated source
    │
    └─ os.WriteFile(ComponentName.generated.go)
```

---

## File Reference

### `compiler.go`

**Public API only.** Contains the single exported function:

```go
func Compile(srcDir string, devMode bool) error
```

Resolves `srcDir` to an absolute path, calls `discoverAndInspectComponents`, builds the `componentMap` used throughout code generation, then calls `compileComponentTemplate` for each discovered component. All other logic is in dedicated files.

---

### `types.go`

**Shared data model.** Declares every struct and package-level variable used across the compiler:

- `componentSchema`, `propertyDescriptor`, `methodDescriptor`, `paramDescriptor` — component introspection types.
- `componentInfo`, `compileOptions`, `loopContext`, `textNodePosition` — pipeline types.
- `dataBindingRegex` — matches `{FieldName}` and `{dotted.path}` expressions.
- `ternaryExprRegex` — matches `{ condition ? 'a' : 'b' }` expressions.
- `booleanShorthandRegex` — matches `{condition}` / `{!condition}` used as HTML boolean attributes.
- `standardBooleanAttrs` — set of HTML attributes that are boolean (no value needed).
- `problematicHTMLTags` — tags that cause noise when emitted by `net/html` (e.g. `<html>`, `<body>`).

Nothing in this file has side effects; it is safe to import anywhere.

---

### `preprocessor.go`

**Template rewriting before HTML parsing.**

The Go `net/html` parser does not understand nojs template directives. This file transforms the raw HTML string before parsing so that directives become valid HTML elements that the rest of the pipeline can traverse.

| Function | What it does |
|---|---|
| `preprocessConditionals(src, path)` | Rewrites `{@if expr}…{@else if}…{@else}…{@/if}` blocks into `<go-conditional><go-if>…</go-if><go-else>…</go-else></go-conditional>` markup |
| `preprocessFor(src, path)` | Rewrites `{@for i, item := range Items}…{@/for}` blocks into `<go-for data-range="Items" …>…</go-for>` markup |

Both functions return errors with file path and approximate line numbers when the syntax is malformed.

---

### `helpers.go`

**Shared utilities.** Functions used by two or more other files:

| Function | Purpose |
|---|---|
| `estimateLineNumber(src, needle)` | Finds the 1-based line of the first occurrence of `needle` in `src` |
| `estimateTextNodeLineNumber(src, text)` | Variant optimised for text nodes |
| `estimateComponentTagLineNumber(src, tagName)` | Variant optimised for component opening tags |
| `getSourceLine(src, line)` | Returns the content of a specific line |
| `getContextLines(src, line, ctx)` | Returns `ctx` lines of context around `line` for error messages |
| `getAvailableFieldNames(comp)` | Returns sorted slice of all prop + state field names |
| `getAvailableMethodNames(comp)` | Returns sorted slice of all method names |
| `findEventLineNumber(n, event, src)` | Locates the line of a specific event attribute on an HTML node |
| `findBody(doc)` | Walks the `*html.Node` tree to find the `<body>` element |
| `findFirstElementChild(n)` | Returns the first `ElementNode` child of `n` |
| `childCount(n)` | Counts element children of `n` |

---

### `validator.go`

**Compile-time semantic validation.** Detects errors early with developer-friendly messages that include file, line, and suggestions.

| Function | Purpose |
|---|---|
| `validateComponentName(name, map, comp, path, line)` | Errors if a PascalCase tag has no matching component; suggests similar names |
| `isBooleanAttribute(attr)` | Returns true for standard HTML boolean attributes |
| `validateBooleanCondition(expr, comp, path, line, src)` | Validates `{field}` used as a boolean attribute exists on the component |
| `validateEventHandler(event, handler, tag, comp, path, line, src)` | Validates `@event="Handler"` — method must exist with the correct signature |
| `levenshteinDistance(a, b)` | Edit-distance implementation used by fuzzy matching |
| `findSimilarComponents(name, map)` | Returns component names within edit-distance 2 of `name` |
| `generateMissingComponentError(name, map, comp, src, path, line)` | Builds the full error message string for unknown component tags |

---

### `discovery.go`

**Filesystem scan and Go AST inspection.**

| Function | Purpose |
|---|---|
| `discoverAndInspectComponents(rootDir)` | Walks `rootDir` recursively for `*.gt.html` files; loads Go packages for each directory; returns `[]componentInfo` |
| `collectUsedComponents(root, map, current)` | Walks the parsed HTML tree to find cross-package component references; returns import paths |
| `inspectGoFile(path, structName)` | Parses a single `.go` file and delegates to `inspectStructInFile` |
| `inspectStructInFile(file, fset, structName, dir)` | Uses `go/ast` to read struct fields, identify props vs state (by naming convention), and collect method signatures |
| `extractTypeName(expr)` | Converts a `go/ast` type expression to a string (e.g. `"[]*vdom.VNode"`) |
| `extractParams(list, fset)` | Converts a `go/ast` parameter list to `[]paramDescriptor` |
| `extractReturns(list)` | Converts a `go/ast` return list to `[]string` |

**Prop vs State convention:** fields whose names match a method name (case-insensitive) are treated as state; all other exported fields are treated as props. Fields of type `[]*vdom.VNode` are identified as the content slot.

---

### `typeresolver.go`

**Nested field type resolution.** Resolves dotted expressions like `{Ctx.Title}` by following the Go type chain across files.

| Function | Purpose |
|---|---|
| `resolveNestedFieldType(parts, comp, dir)` | Resolves a `[]string` field path to its final Go type string |
| `resolvePackageFromAlias(alias, comp)` | Looks up the import path for a package alias used in the struct's file |
| `isBuiltinType(t)` | Returns true for Go primitive types (`string`, `int`, `bool`, …) |
| `findStructFieldType(pkgPath, structName, fieldName)` | Loads a package via `go/packages` and finds a field's type |
| `findStructFieldTypeInDir(dir, structName, fieldName)` | Directory-scoped variant using `go/parser` (avoids full module load) |
| `findPackageDir(importPath)` | Resolves an import path to an absolute directory using `go/packages` |
| `getAvailableNestedFields(parts, comp, dir)` | Returns field names reachable at a dotted path (for error suggestions) |
| `getStructFields(pkgPath, structName)` | Returns all field names of a struct in a package |

---

### `codegen_attributes.go`

**Attribute map and struct literal generation.**

| Function | Purpose |
|---|---|
| `generateAttributesMap(n, receiver, comp, src)` | Produces the Go `map[string]string` literal for an HTML element's attributes, handling `@event`, `{binding}`, ternary, and boolean attributes |
| `generateTernaryExpression(match, receiver, comp)` | Converts a `{ cond ? 'a' : 'b' }` match to a Go ternary expression |
| `generateStructLiteral(n, compInfo, receiver, map, current, src, path, opts, loopCtx)` | Generates the `{Prop: value, …}` struct literal used when rendering a child component |
| `extractOriginalAttributesWithLineNumber(n, src)` | Returns attributes paired with their source line numbers (for error messages) |
| `convertPropValue(raw, goType, receiver, current, src, lineNum, loopCtx)` | Converts a raw attribute value string to a Go expression of the correct type |

---

### `codegen_text.go`

**Text node and slot content generation.**

| Function | Purpose |
|---|---|
| `generateTextExpression(content, receiver, comp, src, line, loopCtx)` | Converts a text node's content to a Go string expression, handling `{binding}`, ternary, and static strings; validates field references |
| `generateSlotTextNodeError(pos, currentComp, src)` | Builds a compile-time error message when a plain text node appears directly inside a slot |
| `collectSlotChildren(n, receiver, map, current, src, opts)` | Walks a component's children to build the `[]*vdom.VNode` slice passed as slot content |

---

### `codegen_loops.go`

**`{@for}` loop code generation.**

| Function | Purpose |
|---|---|
| `generateForLoopCode(n, receiver, map, current, src, opts)` | Generates an IIFE (`func() []*vdom.VNode { … }()`) containing a `for` loop over the bound slice; produces `[]*vdom.VNode` to be spread into the parent element's children |
| `extractTrackByFromParent(n)` | Reads the `data-trackby` attribute emitted by `preprocessFor` to extract the expression used as the VNode key |

Generated loops follow this pattern:
```go
func() []*vdom.VNode {
    var items []*vdom.VNode
    for i, item := range c.Items {
        _ = i
        items = append(items, /* child vnode */)
    }
    return items
}()...
```

---

### `codegen_conditionals.go`

**`{@if}/{@else if}/{@else}` code generation.**

| Function | Purpose |
|---|---|
| `generateConditionalCode(n, receiver, map, current, src, opts, loopCtx)` | Walks the `<go-conditional>` subtree produced by the preprocessor and generates a Go `if / else if / else` expression returning `*vdom.VNode` (or `nil` for the absent branch) |

The generated pattern is:
```go
func() *vdom.VNode {
    if c.IsLoggedIn {
        return /* VNode for true branch */
    } else {
        return nil
    }
}()
```

---

### `codegen_nodes.go`

**Central node dispatch.** `generateNodeCode` is the recursive heart of the code generator. It receives a single `*html.Node` and returns the Go expression string for that node.

| Node type | Action |
|---|---|
| `html.TextNode` | Calls `generateTextExpression`; wraps result in `vdom.Text(…)` |
| `<go-conditional>` | Delegates to `generateConditionalCode` |
| `<go-for>` | Delegates to `generateForLoopCode` |
| ComponentTag (PascalCase) | Validates component exists; calls `generateStructLiteral`; emits `r.RenderChild("key", &Comp{…})` |
| Unknown PascalCase tag | Calls `generateMissingComponentError` and `os.Exit(1)` |
| Standard HTML elements | Calls `generateAttributesMap`; recurses into children; emits the appropriate `vdom.*` helper or `vdom.NewVNode(…)` call |

Also contains:
- `isComponentTag(name)` — returns true when the first character is uppercase.
- `findOriginalTagName(n, lowercase, src)` — recovers the original casing from the HTML source (since `net/html` lowercases all tag names).

---

### `codegen.go`

**Top-level template pipeline.**

| Function | Purpose |
|---|---|
| `compileComponentTemplate(comp, map, inDir, opts)` | Orchestrates the full compile cycle for one component: read → preprocess → parse → generate → format → write |
| `generateApplyPropsBody(comp)` | Produces the sorted assignment statements for `ApplyProps` — copies props in deterministic order, includes the slot field last |

The generated file header includes import suppression lines (`_ = fmt.Sprintf`, `_ = events.AdaptNoArgEvent`, etc.) so that `gofmt`/`go build` do not fail when a component uses none of the standard imports.
