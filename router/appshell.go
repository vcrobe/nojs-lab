//go:build js || wasm

package router

import (
	"fmt"

	"github.com/vcrobe/nojs/console"
	"github.com/vcrobe/nojs/runtime"
	"github.com/vcrobe/nojs/vdom"
)

// AppShell is a stable root component that holds persistent layouts (app shell)
// and swaps only the BodyContent slot when navigation occurs. This preserves
// layout instances and their internal state across navigations including sublayouts.
type AppShell struct {
	runtime.ComponentBase

	// persistent layout instance (app shell)
	persistentLayout runtime.Component

	// current chain of component instances (all from router, volatile)
	currentChain []runtime.Component
	currentKey   string
}

// NewAppShell creates a new AppShell with the given persistent layout component.
// The layout should implement the slot convention: a BodyContent []*vdom.VNode field.
func NewAppShell(persistentLayout runtime.Component) *AppShell {
	return &AppShell{
		persistentLayout: persistentLayout,
		currentChain:     make([]runtime.Component, 0),
	}
}

// SetPage replaces the volatile chain of component instances and triggers a re-render.
// The chain includes components from the router (from pivot onwards).
// When pivot > 0, the chain doesn't include the persistent layout (it's preserved).
func (a *AppShell) SetPage(chain []runtime.Component, key string) {
	console.Log("[AppShell.SetPage] Called with", len(chain), "components, key:", key)
	if len(chain) > 0 {
		console.Log("[AppShell.SetPage] First component type:", fmt.Sprintf("%T", chain[0]))
	}

	// If the chain doesn't include persistentLayout at index 0, prepend it
	// (this happens when pivot > 0 and layouts are preserved)
	if len(chain) == 0 || chain[0] != a.persistentLayout {
		console.Log("[AppShell.SetPage] Prepending persistentLayout to chain")
		fullChain := make([]runtime.Component, 0, len(chain)+1)
		fullChain = append(fullChain, a.persistentLayout)
		fullChain = append(fullChain, chain...)
		a.currentChain = fullChain
	} else {
		a.currentChain = chain
	}
	a.currentKey = key

	console.Log("[AppShell.SetPage] Calling StateHasChanged")
	a.StateHasChanged()
}

// Render composes the persistent layout with the current component chain.
func (a *AppShell) Render(r runtime.Renderer) *vdom.VNode {
	console.Log("[AppShell.Render] Called, chain length:", len(a.currentChain))

	type rendererSetter interface {
		SetRenderer(runtime.Renderer)
	}

	// Ensure persistent layout has renderer
	if a.persistentLayout != nil {
		if rs, ok := interface{}(a.persistentLayout).(rendererSetter); ok {
			rs.SetRenderer(r)
		}
	}

	// Link the chain: inject each child into parent's BodyContent slot
	var slotChildren []*vdom.VNode
	if len(a.currentChain) > 0 {
		console.Log("[AppShell.Render] Processing chain")
		chainIndex := 0
		if a.currentChain[0] == a.persistentLayout {
			console.Log("[AppShell.Render] First component is persistentLayout, skipping")
			chainIndex = 1
		}

		// Link bottom-up: leaf → root
		for i := len(a.currentChain) - 1; i > chainIndex; i-- {
			child := a.currentChain[i]
			parent := a.currentChain[i-1]

			if rs, ok := interface{}(child).(rendererSetter); ok {
				rs.SetRenderer(r)
			}

			slotKey := fmt.Sprintf("slot-chain-%d-%T-%p", i, child, child)
			childVNode := r.RenderChild(slotKey, child)
			if childVNode != nil {
				console.Log("[AppShell.Render] Linking", fmt.Sprintf("%T", child), "into", fmt.Sprintf("%T", parent))
				if layout, ok := parent.(interface{ SetBodyContent([]*vdom.VNode) }); ok {
					layout.SetBodyContent([]*vdom.VNode{childVNode})
				}
			}
		}

		// Render the first non-layout component in the chain
		if chainIndex < len(a.currentChain) {
			rootComponent := a.currentChain[chainIndex]
			console.Log("[AppShell.Render] Rendering root component at index", chainIndex, "type:", fmt.Sprintf("%T", rootComponent))

			if rs, ok := interface{}(rootComponent).(rendererSetter); ok {
				rs.SetRenderer(r)
			}

			slotKey := fmt.Sprintf("slot-root-%T-%p", rootComponent, rootComponent)
			childVNode := r.RenderChild(slotKey, rootComponent)
			if childVNode != nil {
				slotChildren = []*vdom.VNode{childVNode}
			}
		}
	}

	// Inject into layout's BodyContent slot
	if a.persistentLayout != nil {
		if layout, ok := a.persistentLayout.(interface{ SetBodyContent([]*vdom.VNode) }); ok {
			layout.SetBodyContent(slotChildren)
		}
		return r.RenderChild("persistent-layout", a.persistentLayout)
	}

	// Fallback: render the first non-layout component from chain
	if len(a.currentChain) > 0 {
		chainIndex := 0
		if a.currentChain[0] == a.persistentLayout {
			chainIndex = 1
		}
		if chainIndex < len(a.currentChain) {
			rootComponent := a.currentChain[chainIndex]
			slotKey := fmt.Sprintf("slot-root-%T-%p", rootComponent, rootComponent)
			return r.RenderChild(slotKey, rootComponent)
		}
	}

	return vdom.NewVNode("div", nil, nil, "")
}
