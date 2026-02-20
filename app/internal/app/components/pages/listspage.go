//go:build js || wasm

package pages

import (
	"github.com/vcrobe/nojs/runtime"
)

var techPool = []string{
	"Go", "Rust", "WebAssembly", "TypeScript",
	"Zig", "C++", "Kotlin", "Swift", "Python", "Elixir",
}

// ListsPage demonstrates {@for} list rendering with trackBy.
type ListsPage struct {
	runtime.ComponentBase

	Items       []string
	NextIndex   int
	HasItems    bool
	RenderCount int
}

func (c *ListsPage) OnMount() {
	c.Items = []string{"Go", "Rust", "WebAssembly"}
	c.NextIndex = 3
	c.HasItems = true
}

func (c *ListsPage) OnParametersSet() {
	c.RenderCount++
}

func (c *ListsPage) AddItem() {
	if c.NextIndex < len(techPool) {
		c.Items = append(c.Items, techPool[c.NextIndex])
		c.NextIndex++
		c.HasItems = true
		c.StateHasChanged()
	}
}

func (c *ListsPage) RemoveLast() {
	if len(c.Items) > 0 {
		c.Items = c.Items[:len(c.Items)-1]
		c.HasItems = len(c.Items) > 0
		c.StateHasChanged()
	}
}

func (c *ListsPage) Reset() {
	c.Items = []string{"Go", "Rust", "WebAssembly"}
	c.NextIndex = 3
	c.HasItems = true
	c.StateHasChanged()
}
