//go:build js || wasm
// +build js wasm

package main

import (
	"github.com/vcrobe/nojs/vdom"
)

// TestComponent is a sample framework component that demonstrates
// the virtual DOM structure with various elements and interactivity.
type TestComponent struct {
	instanceName string
	clickCount   int
}

// NewTestComponent creates a new instance of TestComponent.
func NewTestComponent(instanceName string, clickCount int) *TestComponent {
	return &TestComponent{
		instanceName: instanceName,
		clickCount:   clickCount,
	}
}

// handleButtonClick is the event handler for button clicks.
func (tc *TestComponent) handleButtonClick() {
	tc.clickCount++
	println("Button clicked from instance `", tc.instanceName, "`. Click count:", tc.clickCount)
}

// Render implements the Component interface and returns the virtual DOM structure.
func (tc *TestComponent) Render() *vdom.VNode {
	return vdom.Div(map[string]any{"id": "test-div", "data-attr": -1.3},
		vdom.Paragraph(tc.instanceName, nil),
		vdom.InputText(map[string]any{"placeholder": "Type here...", "id": "test-input"}),
		vdom.Button("Click me", map[string]any{
			"onClick": func() { tc.handleButtonClick() },
		}, nil),
	)
}
