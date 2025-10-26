# Component Lifecycle Methods

Components can hook into specific moments in their lifecycle by implementing optional interfaces defined in the `runtime` package. The `runtime.ComponentBase` struct provides the necessary infrastructure for these hooks to function correctly with the manual refresh system.

## Available Lifecycle Hooks

### 1. Initialization (`OnInit`)

- **Interface:** `runtime.Initializer`
- **Method:** `OnInit()`
- **When:** Called **once** by the runtime after the component instance is created, but _before_ the first render.
- **Purpose:** Ideal for one-time setup tasks or initiating asynchronous operations like data fetching.

**Example:**

```go
type UserProfile struct {
    runtime.ComponentBase
    UserID    int
    User      *UserData
    IsLoading bool
}

// Ensure the component implements the Initializer interface
var _ runtime.Initializer = (*UserProfile)(nil)

func (c *UserProfile) OnInit() {
    println("UserProfile initialized")
    c.IsLoading = true
    go c.fetchUserData()
}
```

### 2. Parameter Updates (`OnPropertiesSet`)

- **Interface:** `runtime.ParameterReceiver`
- **Method:** `OnPropertiesSet()`
- **When:** Called by the runtime _every time_ the component receives parameters (props) from its parent, _just before_ the `Render` method is called. This includes the initial render.
- **Purpose:** Allows the component to react to changes in its input properties, for example, by fetching new data if an ID prop has changed.

**Example:**

```go
type DataDisplay struct {
    runtime.ComponentBase
    DataID     int
    prevDataID int  // Track previous value for change detection
    Data       *DataModel
}

// Ensure the component implements the ParameterReceiver interface
var _ runtime.ParameterReceiver = (*DataDisplay)(nil)

func (c *DataDisplay) OnPropertiesSet() {
    // Manual change detection - only fetch if DataID changed
    if c.DataID != c.prevDataID {
        c.prevDataID = c.DataID
        println("DataID changed to", c.DataID)
        go c.fetchData()
    }
}
```

### 3. Cleanup (`OnDestroy`)

- **Interface:** `runtime.Cleaner`
- **Method:** `OnDestroy()`
- **When:** Called **once** by the runtime when the component instance is removed from the component tree (unmounted).
- **Purpose:** Essential for cleaning up resources like goroutines, timers, event listeners, and `js.Func` callbacks to prevent memory leaks and unwanted side effects.

**Example:**

```go
type TimerComponent struct {
    runtime.ComponentBase
    ctx    context.Context    `nojs:"state"`
    cancel context.CancelFunc `nojs:"state"`
    Count  int                `nojs:"state"`
}

// Ensure the component implements both Initializer and Cleaner interfaces
var _ runtime.Initializer = (*TimerComponent)(nil)
var _ runtime.Cleaner = (*TimerComponent)(nil)

func (c *TimerComponent) OnInit() {
    c.ctx, c.cancel = context.WithCancel(context.Background())
    go c.startTimer()
}

func (c *TimerComponent) startTimer() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-c.ctx.Done():
            println("Timer cleanup complete")
            return
        case <-ticker.C:
            c.Count++
            c.StateHasChanged()
        }
    }
}

func (c *TimerComponent) OnDestroy() {
    if c.cancel != nil {
        c.cancel() // Stop the goroutine gracefully
    }
}
```

## Handling Asynchronous Operations

Lifecycle methods like `OnInit` and `OnPropertiesSet` are **synchronous**. To perform long-running tasks like API calls without blocking the UI, you must launch a goroutine. The `StateHasChanged()` method (provided by embedding `runtime.ComponentBase`) is essential for updating the UI when the async task completes.

### Pattern for Async Data Fetching

1. **Launch Goroutine:** Start the asynchronous task in a separate goroutine from within the lifecycle method (e.g., `go c.fetchData()`).
2. **Initial Render (Loading State):** The lifecycle method returns immediately, allowing the component to render its initial state (e.g., a "Loading..." message).
3. **Update State & Refresh:** When the asynchronous task completes within the goroutine, update the component's state fields and then **call `c.StateHasChanged()`** to trigger a UI re-render with the new data.

### Complete Example

```go
package appcomponents

import (
    "time"
    "github.com/vcrobe/nojs/runtime"
)

type UserData struct {
    Name  string
    Email string
}

type UserProfile struct {
    runtime.ComponentBase  // Embed the base component
    UserID    int         // Prop: User ID to fetch
    User      *UserData   // State: Fetched user data
    IsLoading bool        // State: Loading indicator
}

// Ensure UserProfile implements Initializer
var _ runtime.Initializer = (*UserProfile)(nil)

func (c *UserProfile) OnInit() {
    c.IsLoading = true
    go c.fetchUserData() // Launch async task
}

func (c *UserProfile) fetchUserData() {
    // Simulate API call delay
    time.Sleep(2 * time.Second)
    
    // Simulate fetched data
    fetchedUser := &UserData{
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    // Update state
    c.User = fetchedUser
    c.IsLoading = false
    
    // Trigger re-render (safe to call from goroutine)
    c.StateHasChanged()
}
```

**Template (`UserProfile.gt.html`):**

```html
<div>
    {@if IsLoading}
        <p>Loading user profile...</p>
    {@else}
        <h1>{User.Name}</h1>
        <p>Email: {User.Email}</p>
    {@endif}
</div>
```

## Lifecycle Order

For a component instance, the lifecycle methods are called in this order:

1. **Component created** (struct instantiated by parent)
2. **`OnInit()`** _(only on first render)_
3. **`OnPropertiesSet()`** _(on every render, including first)_
4. **`Render()`** _(generates VDOM tree)_

On subsequent renders when props change:

1. **Props updated** (via generated `ApplyProps` method)
2. **`OnPropertiesSet()`**
3. **`Render()`**

When a component is unmounted (removed from the tree):

1. **Component detected as no longer in render tree**
2. **`OnDestroy()`** _(cleanup resources)_
3. **Component instance removed from memory**

### Cleanup Order

When multiple components are unmounted in a single render cycle, `OnDestroy` is called in a **depth-first** manner, cleaning up **children before parents**. This ensures that child components can properly clean up before their parent's cleanup runs.

## Prop Updates and Instance Preservation

The framework automatically preserves component instances across renders to maintain internal state. When a parent re-renders with updated props:

- The **existing component instance is reused** (preserving state like counters, timers, etc.)
- New prop values are **applied automatically** via the compiler-generated `ApplyProps` method
- `OnPropertiesSet` is called to allow the component to react to prop changes

**Note:** You don't need to manually implement `ApplyProps` - the compiler generates it automatically for each component based on its exported fields.

### **Props vs State: Using Struct Tags**

To distinguish between **props** (values passed by parent) and **state** (internal component data), the framework uses Go struct tags and visibility rules:

| Field Type | In ApplyProps? | In Template? | Example |
|------------|---------------|--------------|---------|
| **Unexported field** | ❌ No | ❌ No (unless via method) | `clickCount int` |
| **Exported field (no tag)** | ✅ Yes | ✅ Yes | `Title string` |
| **Exported field with `nojs:"state"`** | ❌ No | ✅ Yes | `IsLoading bool \`nojs:"state"\`` |

**Example:**

```go
type DataDisplay struct {
    runtime.ComponentBase
    
    // Props - can be set by parent, copied in ApplyProps
    DataID int
    Title  string
    
    // State - internal only, NOT copied in ApplyProps (tagged with nojs:"state")
    IsLoading  bool   `nojs:"state"`
    ErrorMsg   string `nojs:"state"`
    prevDataID int    `nojs:"state"` // For change detection
    
    // Private state - unexported, not accessible in templates
    apiKey string
}
```

**Why This Matters:**

Without the `nojs:"state"` tag, all exported fields are treated as props. When a parent re-renders, `ApplyProps` copies values from the new instance to the existing instance, which would **overwrite your internal state**. By tagging fields as `nojs:"state"`, you tell the compiler: "This field is internal - don't copy it during prop updates."

**Complete Example with State Management:**

```go
type UserList struct {
    runtime.ComponentBase
    
    // Prop: Can be set by parent
    Title string
    
    // State: Internal, preserved across re-renders
    Users      []User  `nojs:"state"`
    IsLoading  bool    `nojs:"state"`
    ErrorMsg   string  `nojs:"state"`
}

var _ runtime.Initializer = (*UserList)(nil)

func (c *UserList) OnInit() {
    c.IsLoading = true
    go c.loadUsers()
}

func (c *UserList) loadUsers() {
    // Simulate API call
    time.Sleep(1 * time.Second)
    
    c.Users = []User{{Name: "Alice"}, {Name: "Bob"}}
    c.IsLoading = false
    c.StateHasChanged() // Trigger re-render with new state
}

func (c *UserList) AddUser() {
    c.Users = append(c.Users, User{Name: "New User"})
    c.StateHasChanged()
}
```

**Template (`UserList.gt.html`):**

```html
<div>
    <h2>{Title}</h2>
    {@if IsLoading}
        <p>Loading users...</p>
    {@else}
        {@for user in Users}
            <p>{user.Name}</p>
        {@endfor}
        <button @onclick="AddUser">Add User</button>
    {@endif}
</div>
```

When the parent updates the `Title` prop, only `Title` is copied via `ApplyProps`. The `Users`, `IsLoading`, and `ErrorMsg` state fields remain unchanged, preserving the component's internal data.

## Development vs Production Modes

The framework supports two build modes that affect lifecycle error handling:

### Development Mode

Build with the `dev` tag:

```bash
GOOS=js GOARCH=wasm go build -tags=dev -o main.wasm
```

**Behavior:** Panics in `OnInit` and `OnPropertiesSet` propagate immediately, causing the application to crash. This helps you catch bugs during development.

### Production Mode

Build without tags:

```bash
GOOS=js GOARCH=wasm go build -o main.wasm
```

**Behavior:** Panics in `OnInit` and `OnPropertiesSet` are recovered and logged to the console, preventing application crashes.

## Best Practices

1. **Keep lifecycle methods fast:** Avoid heavy computation in `OnInit` or `OnPropertiesSet`. Launch goroutines for async work.

2. **Always call `StateHasChanged()` after async updates:** The framework doesn't automatically detect state changes from goroutines.

3. **Implement change detection in `OnPropertiesSet`:** Compare current props with previous values to avoid unnecessary work:

   ```go
   func (c *DataDisplay) OnPropertiesSet() {
       if c.DataID != c.prevDataID {
           c.prevDataID = c.DataID
           go c.fetchData()
       }
   }
   ```

4. **Use compile-time checks:** Add interface assertions to catch missing implementations:

   ```go
   var _ runtime.Initializer = (*UserProfile)(nil)
   var _ runtime.ParameterReceiver = (*UserProfile)(nil)
   var _ runtime.Cleaner = (*UserProfile)(nil)
   ```

5. **Mark internal state with `nojs:"state"` tag:** Prevent state fields from being overwritten by `ApplyProps`:

   ```go
   type DataDisplay struct {
       runtime.ComponentBase
       
       DataID    int    // Prop - can be updated by parent
       IsLoading bool   `nojs:"state"` // State - internal only
       Data      []Item `nojs:"state"` // State - internal only
   }
   ```

6. **Initialize loading states:** Set loading flags in `OnInit` before launching async tasks so the initial render shows a loading indicator.

7. **Always clean up resources in `OnDestroy`:** Use context cancellation for goroutines, release `js.Func` callbacks, and close connections:

   ```go
   func (c *Component) OnDestroy() {
       if c.cancel != nil {
           c.cancel() // Stop goroutines
       }
       if c.jsCallback.Type() != js.TypeUndefined {
           c.jsCallback.Release() // Free JS callback
       }
   }
   ```

8. **Use context for goroutine lifecycle:** Pattern goroutines with context cancellation so they can be cleanly stopped:

   ```go
   func (c *Component) OnInit() {
       c.ctx, c.cancel = context.WithCancel(context.Background())
       go c.backgroundTask()
   }
   
   func (c *Component) backgroundTask() {
       for {
           select {
           case <-c.ctx.Done():
               return // Exit when context is cancelled
           case <-time.After(1 * time.Second):
               // Do work...
           }
       }
   }
   ```

9. **Keep `OnDestroy` synchronous:** Don't block in cleanup methods. Use fire-and-forget for network operations, or use browser APIs like `sendBeacon` for critical data.

## Compiler Flag

The AOT compiler supports a `-dev` flag that enables development mode features:

```bash
./compiler/aotcompiler -in appcomponents -out appcomponents -dev
```

This flag controls:
- Development warnings (e.g., empty slice warnings in `{@for}` loops)
- Verbose error messages
- Empty slot content warnings

The `-dev` flag is separate from build tags and affects code generation, not runtime behavior.

## Common Patterns

### Pattern 1: Fetch Data on Mount

```go
func (c *Component) OnInit() {
    c.IsLoading = true
    go c.fetchData()
}
```

### Pattern 2: React to Prop Changes

```go
func (c *Component) OnPropertiesSet() {
    if c.UserID != c.prevUserID {
        c.prevUserID = c.UserID
        c.IsLoading = true
        go c.fetchUserData()
    }
}
```

**Note:** Mark `prevUserID` with `nojs:"state"` to prevent it from being overwritten by `ApplyProps`.

### Pattern 3: Distinguish Props from State

```go
type MyComponent struct {
    runtime.ComponentBase
    
    // Props (no tag) - set by parent
    Title  string
    UserID int
    
    // State (with tag) - internal only
    IsLoading  bool   `nojs:"state"`
    Data       []Item `nojs:"state"`
    errorMsg   string // unexported = private state
}
```

### Pattern 4: Setup with Cleanup

```go
type WebSocketComponent struct {
    runtime.ComponentBase
    ctx       context.Context    `nojs:"state"`
    cancel    context.CancelFunc `nojs:"state"`
    Messages  []string           `nojs:"state"`
}

var _ runtime.Initializer = (*WebSocketComponent)(nil)
var _ runtime.Cleaner = (*WebSocketComponent)(nil)

func (c *WebSocketComponent) OnInit() {
    c.ctx, c.cancel = context.WithCancel(context.Background())
    go c.listenToWebSocket()
}

func (c *WebSocketComponent) listenToWebSocket() {
    for {
        select {
        case <-c.ctx.Done():
            println("WebSocket listener stopped")
            return
        default:
            // Read from WebSocket...
            c.Messages = append(c.Messages, "New message")
            c.StateHasChanged()
        }
    }
}

func (c *WebSocketComponent) OnDestroy() {
    if c.cancel != nil {
        c.cancel() // Stop the listener goroutine
    }
}
```

### Pattern 5: Browser Event Listeners

```go
type WindowListener struct {
    runtime.ComponentBase
    resizeCallback js.Func `nojs:"state"`
    Width          int     `nojs:"state"`
}

var _ runtime.Initializer = (*WindowListener)(nil)
var _ runtime.Cleaner = (*WindowListener)(nil)

func (c *WindowListener) OnInit() {
    c.resizeCallback = js.FuncOf(func(this js.Value, args []js.Value) any {
        c.Width = js.Global().Get("window").Get("innerWidth").Int()
        c.StateHasChanged()
        return nil
    })
    
    js.Global().Get("window").Call("addEventListener", "resize", c.resizeCallback)
}

func (c *WindowListener) OnDestroy() {
    js.Global().Get("window").Call("removeEventListener", "resize", c.resizeCallback)
    c.resizeCallback.Release() // Critical: free the Go callback
}
```

### Pattern 6: Batch Analytics on Unmount

```go
type AnalyticsTracker struct {
    runtime.ComponentBase
    events []Event `nojs:"state"`
}

var _ runtime.Cleaner = (*AnalyticsTracker)(nil)

func (c *AnalyticsTracker) TrackEvent(event Event) {
    c.events = append(c.events, event)
}

func (c *AnalyticsTracker) OnDestroy() {
    if len(c.events) > 0 {
        // Use sendBeacon for guaranteed delivery even during page unload
        data := js.ValueOf(map[string]any{
            "events": c.events,
        })
        js.Global().Get("navigator").Call("sendBeacon", "/api/events", data)
    }
}
```

## See Also

- [Conditional Rendering](CONDITIONAL_RENDERING.md)
- [List Rendering](LIST_RENDERING.md)
- [Content Projection](CONTENT_PROJECTION.md)
