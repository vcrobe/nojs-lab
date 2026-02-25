# Signals

Signals are reactive, thread-safe values that live **outside the component tree**. They survive route transitions, component unmounts, and re-renders — making them the correct tool for any state that must persist across the component lifecycle.

> **Note — future relocation:** `Signal[T]` is a general-purpose reactive primitive that has no dependency on the framework's runtime, VDOM, or renderer. It currently lives in `github.com/ForgeLogic/nojs/signals` for convenience, but it will be extracted into its own standalone module once the project's module structure matures. No framework APIs depend on this package, so the move will not affect any framework internals — only the import path in your application code will change.

---

## Table of Contents

1. [Why signals exist](#1-why-signals-exist)
2. [Signal vs. component state](#2-signal-vs-component-state)
3. [API reference](#3-api-reference)
   - [NewSignal](#newsignal)
   - [Get](#get)
   - [Set](#set)
   - [Subscribe / Unsubscribe](#subscribe--unsubscribe)
4. [Defining signals](#4-defining-signals)
5. [Reading and writing a signal](#5-reading-and-writing-a-signal)
6. [Subscribing to changes](#6-subscribing-to-changes)
7. [Thread safety](#7-thread-safety)

---

## 1. Why signals exist

Component state (fields on a struct) is **ephemeral**: the router destroys a component instance when navigating away and creates a fresh one when navigating back. Any state held in component fields is lost on that transition.

Signals solve this by owning state at the **application level**, completely independent of any component instance. A component can read a signal when it mounts, write to it when something changes, and any other subscriber is notified immediately — all without coordination through props or the renderer.

---

## 2. Signal vs. component state

| | Component field | Signal |
|---|---|---|
| Lifetime | Same as the component instance | Application lifetime |
| Survives route change | No — reset to zero value | Yes |
| Shared across components | Only via prop drilling | Yes — any component can read/write |
| Triggers re-render | Only in the owning component (via `StateHasChanged`) | Subscriber callbacks fire; each subscriber decides whether to re-render |
| Thread safe | No guarantee | Yes — `sync.RWMutex` guarded |

Use a signal when:
- The value must survive navigation or component remounting.
- Two or more unrelated components need to share the same piece of state.
- You want reactive propagation without coupling components to each other.

Use a plain component field when:
- The value is local and private to a single component.
- It is safe to reset to its zero value every time the component is mounted.

---

## 3. API reference

### NewSignal

```go
func NewSignal[T any](initial T) *Signal[T]
```

Creates a new signal with an initial value. `T` can be any type — primitive, struct, slice, etc.

```go
import "github.com/ForgeLogic/nojs/signals"

var Count    = signals.NewSignal(0)
var Username = signals.NewSignal("")
var Items    = signals.NewSignal([]string{})
```

---

### Get

```go
func (s *Signal[T]) Get() T
```

Returns the current value. Safe to call concurrently from multiple goroutines.

```go
current := appstate.Count.Get()
```

---

### Set

```go
func (s *Signal[T]) Set(v T)
```

Replaces the current value and synchronously calls every registered subscriber. The lock is released before subscribers are called, so subscribers may themselves call `Get` or `Set` without deadlocking.

```go
appstate.Count.Set(appstate.Count.Get() + 1)
```

---

### Subscribe / Unsubscribe {#subscribe--unsubscribe}

```go
func (s *Signal[T]) Subscribe(fn func()) (unsubscribe func())
```

Registers a callback that fires after every `Set` call. Returns an **unsubscribe function** — store it and call it in `OnUnmount` to prevent the signal from holding a reference to a destroyed component.

```go
unsub := appstate.Count.Subscribe(func() {
    c.count = appstate.Count.Get()
    c.StateHasChanged()
})

// later, in OnUnmount:
unsub()
```

---

## 4. Defining signals

Signals are application-owned, not framework-owned. The `Signal[T]` type is provided by `github.com/ForgeLogic/nojs/signals`. Declare your app-level signal variables in a dedicated package (e.g. `appstate`) so any component can import them.

```
nojs/
  signals/
    signals.go    ← Signal[T] implementation (github.com/ForgeLogic/nojs/signals)

app/
  internal/
    appstate/
      appstate.go   ← declare your global signals here
```

```go
// appstate/appstate.go
package appstate

import "github.com/ForgeLogic/nojs/signals"

// Add new global signals here as the app grows.
var RenderCount = signals.NewSignal(1)
var NextIDIndex = signals.NewSignal(1)
```

Keep each signal declaration close to a comment explaining what it represents and what owns/writes to it.

---

## 5. Reading and writing a signal

The most common pattern is **reading on mount and writing on user action**:

```go
// pages/routerparamspage.go
//go:build js || wasm

package pages

import (
    "github.com/ForgeLogic/app/internal/appstate"
    "github.com/ForgeLogic/nojs/runtime"
)

type RouterParamsPage struct {
    runtime.ComponentBase

    ID          string
    RenderCount int // local copy, refreshed on mount
}

// OnParametersSet fires every time the router mounts or re-parameterises this
// component. Read the signal here so the local copy is always fresh.
func (c *RouterParamsPage) OnParametersSet() {
    c.RenderCount = appstate.RenderCount.Get()
}

// GoToNext is called from a button click in the template.
func (c *RouterParamsPage) GoToNext() {
    // Write the signal — the new value is immediately visible to all readers.
    appstate.RenderCount.Set(appstate.RenderCount.Get() + 1)

    nextIdx := appstate.NextIDIndex.Get()
    next := demoIDs[nextIdx%len(demoIDs)]
    appstate.NextIDIndex.Set((nextIdx + 1) % len(demoIDs))

    c.Navigate("/demo/router/" + next)
}
```

> The component imports from its own `appstate` package, not directly from `github.com/ForgeLogic/nojs/signals`. This keeps signal declarations centralised and avoids scattering `signals.NewSignal(...)` calls across the codebase.

Key points:
- `OnParametersSet` pulls the latest value from the signal each time the component is (re)mounted — no stale data after navigation.
- `GoToNext` writes to the signal before navigating. When the router creates a fresh `RouterParamsPage` instance for the new route, `OnParametersSet` reads the already-updated value.

---

## 6. Subscribing to changes

Subscribe when a component needs to react to a signal change **without** being remounted by the router (i.e., the component stays alive and must update its view in place).

**Always unsubscribe in `OnUnmount`** — failing to do so keeps a closure referencing the component alive and may cause phantom re-render calls on an already-destroyed component.

```go
import (
    "github.com/ForgeLogic/app/internal/appstate"
    "github.com/ForgeLogic/nojs/runtime"
)

type LiveCounter struct {
    runtime.ComponentBase
    count    int
    unsub    func()
}

func (c *LiveCounter) OnMount() {
    c.count = appstate.Count.Get()

    c.unsub = appstate.Count.Subscribe(func() {
        c.count = appstate.Count.Get()
        c.StateHasChanged() // ask the renderer to re-render this component
    })
}

func (c *LiveCounter) OnUnmount() {
    c.unsub() // remove the subscription to avoid a dangling reference
}
```

---

## 7. Thread safety

`Signal[T]` is safe for concurrent access:

- `Get` acquires a **read lock** (`sync.RWMutex`), so multiple concurrent reads do not block each other.
- `Set` acquires a **write lock**, copies the subscriber slice, releases the lock, then calls each subscriber outside the lock.

Calling `Get` or `Set` from inside a subscriber callback is safe because the lock has been released before subscribers are invoked.

Subscriber callbacks run **synchronously on the same goroutine that called `Set`**. In the browser WASM environment this is always the main JS thread, so no additional synchronisation is needed when updating DOM state from a subscriber.
