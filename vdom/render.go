//go:build js || wasm
// +build js wasm

package vdom

import (
	"syscall/js"

	"github.com/vcrobe/nojs/console"
)

func Clear(selector string) {
	println("Calling ClearSelector(%s)", selector)

	if selector == "" {
		return
	}

	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}

	mount := doc.Call("querySelector", selector)
	if !mount.Truthy() {
		console.Error("Mount element not found for selector:", selector)
		return
	}

	// Set innerHTML to an empty string to clear all children.
	mount.Set("innerHTML", "")
}

// RenderToSelector mounts the VNode under the first element matching the CSS selector.
func RenderToSelector(selector string, n *VNode) {
	if n == nil || selector == "" {
		return
	}

	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}

	mount := doc.Call("querySelector", selector)

	if !mount.Truthy() {
		console.Error("Mount element not found for selector:", selector)
		return
	}

	RenderTo(mount, n)
}

// RenderTo appends the rendered node to a specific mount element.
func RenderTo(mount js.Value, n *VNode) {
	if n == nil {
		return
	}

	el := createElement(n)

	if el.Truthy() {
		mount.Call("appendChild", el)
	}
}

// setAttributeValue sets an attribute on an element, handling boolean attributes correctly.
func setAttributeValue(el js.Value, key string, value interface{}) {
	// Handle boolean attributes
	if boolVal, ok := value.(bool); ok {
		if boolVal {
			// For boolean attributes, set them without a value (or with empty string)
			el.Call("setAttribute", key, "")
		}
		// If false, don't set the attribute at all
		return
	}

	// For all other types, convert to string and set normally
	el.Call("setAttribute", key, value)
}

func createElement(n *VNode) js.Value {
	doc := js.Global().Get("document")
	if !doc.Truthy() || n == nil {
		return js.Undefined()
	}

	switch n.Tag {
	case "p":
		el := doc.Call("createElement", "p")

		if n.Content != "" {
			el.Set("textContent", n.Content)
		}

		if n.Attributes != nil {
			for k, v := range n.Attributes {
				setAttributeValue(el, k, v)
			}
		}

		// children ignored for now
		return el
	case "div":
		el := doc.Call("createElement", "div")

		if n.Attributes != nil {
			for k, v := range n.Attributes {
				setAttributeValue(el, k, v)
			}
		}

		if n.Content != "" {
			el.Set("textContent", n.Content)
		}

		if n.Children != nil {
			for _, child := range n.Children {
				childEl := createElement(child)
				if childEl.Truthy() {
					el.Call("appendChild", childEl)
				}
			}
		}

		return el
	case "input":
		el := doc.Call("createElement", "input")

		if n.Attributes != nil {
			for k, v := range n.Attributes {
				setAttributeValue(el, k, v)
			}
		}

		// For text input, set value if provided in Content
		if n.Content != "" {
			el.Set("value", n.Content)
		}

		return el
	case "button":
		el := doc.Call("createElement", "button")

		if n.Attributes != nil {
			for k, v := range n.Attributes {
				setAttributeValue(el, k, v)
			}
		}

		if n.Content != "" {
			el.Set("textContent", n.Content)
		} else if n.Children != nil {
			for _, child := range n.Children {
				childEl := createElement(child)
				if childEl.Truthy() {
					el.Call("appendChild", childEl)
				}
			}
		}

		// Attach Go OnClick handler if present
		if n.OnClick != nil {
			cb := js.FuncOf(func(this js.Value, args []js.Value) any {
				n.OnClick()
				return nil
			})
			el.Call("addEventListener", "click", cb)
			// Optionally store cb somewhere to release later if needed
		}

		return el

	default:
		console.Error("Unsupported tag: ", n.Tag)
		return js.Undefined()
	}
}
