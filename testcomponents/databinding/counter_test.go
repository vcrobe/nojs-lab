//go:build !wasm
// +build !wasm

package databinding

import (
	"testing"

	"github.com/vcrobe/nojs/testcomponents"
)

// TestDataBinding_InitialRender verifies that data binding correctly
// interpolates component state into the VDOM on the initial render.
func TestDataBinding_InitialRender(t *testing.T) {
	// Arrange: Create a counter component with initial state
	counter := &Counter{
		Count: 5,
		Label: "Test Counter",
	}

	// Attach the test renderer
	renderer := testcomponents.NewTestRenderer(counter)

	// Act: Perform initial render
	vnode := renderer.RenderRoot()

	// Assert: Verify the root structure
	if vnode.Tag != "div" {
		t.Errorf("Expected root tag 'div', got '%s'", vnode.Tag)
	}

	if len(vnode.Children) != 2 {
		t.Fatalf("Expected 2 children (2 paragraphs), got %d", len(vnode.Children))
	}

	// Assert: Verify data binding for Count field
	countParagraph := vnode.Children[0]
	if countParagraph.Tag != "p" {
		t.Errorf("Expected first child to be 'p', got '%s'", countParagraph.Tag)
	}
	expectedCount := "Count: 5"
	if countParagraph.Content != expectedCount {
		t.Errorf("Expected content '%s', got '%s'", expectedCount, countParagraph.Content)
	}

	// Assert: Verify data binding for Label field
	labelParagraph := vnode.Children[1]
	if labelParagraph.Tag != "p" {
		t.Errorf("Expected second child to be 'p', got '%s'", labelParagraph.Tag)
	}
	expectedLabel := "Label: Test Counter"
	if labelParagraph.Content != expectedLabel {
		t.Errorf("Expected content '%s', got '%s'", expectedLabel, labelParagraph.Content)
	}
}

// TestDataBinding_StateUpdate verifies that calling StateHasChanged()
// triggers a re-render with updated data binding values.
func TestDataBinding_StateUpdate(t *testing.T) {
	// Arrange: Create a counter with initial state
	counter := &Counter{
		Count: 3,
		Label: "Initial",
	}

	renderer := testcomponents.NewTestRenderer(counter)
	vnode1 := renderer.RenderRoot()

	// Verify initial state
	if vnode1.Children[0].Content != "Count: 3" {
		t.Errorf("Initial count incorrect: %s", vnode1.Children[0].Content)
	}
	if vnode1.Children[1].Content != "Label: Initial" {
		t.Errorf("Initial label incorrect: %s", vnode1.Children[1].Content)
	}

	// Act: Update component state using Increment method (which calls StateHasChanged)
	counter.Increment()

	// Assert: Verify the VDOM was updated
	vnode2 := renderer.GetCurrentVDOM()

	if vnode2.Children[0].Content != "Count: 4" {
		t.Errorf("Expected 'Count: 4' after increment, got '%s'", vnode2.Children[0].Content)
	}

	// Label should remain unchanged
	if vnode2.Children[1].Content != "Label: Initial" {
		t.Errorf("Label should not change, got '%s'", vnode2.Children[1].Content)
	}
}

// TestDataBinding_MultipleUpdates verifies that multiple state changes
// each trigger re-renders with correct data binding.
func TestDataBinding_MultipleUpdates(t *testing.T) {
	// Arrange
	counter := &Counter{
		Count: 2,
		Label: "Start",
	}

	renderer := testcomponents.NewTestRenderer(counter)
	renderer.RenderRoot()

	// Act & Assert: Multiple increments
	for i := 1; i <= 5; i++ {
		counter.Increment()
		vnode := renderer.GetCurrentVDOM()
		expected := "Count: " + string(rune('2'+i))
		if vnode.Children[0].Content != expected {
			t.Errorf("After %d increment(s), expected '%s', got '%s'",
				i, expected, vnode.Children[0].Content)
		}
	}

	// Act & Assert: Update label
	counter.SetLabel("Updated")
	vnode := renderer.GetCurrentVDOM()

	if vnode.Children[1].Content != "Label: Updated" {
		t.Errorf("Expected 'Label: Updated', got '%s'", vnode.Children[1].Content)
	}

	// Count should remain at 7
	if vnode.Children[0].Content != "Count: 7" {
		t.Errorf("Count should still be 7, got '%s'", vnode.Children[0].Content)
	}
}

// TestDataBinding_RenderIsolation verifies that multiple component instances
// maintain separate state and VDOM.
func TestDataBinding_RenderIsolation(t *testing.T) {
	// Arrange: Create two separate counter instances
	counter1 := &Counter{Count: 10, Label: "First"}
	counter2 := &Counter{Count: 20, Label: "Second"}

	renderer1 := testcomponents.NewTestRenderer(counter1)
	renderer2 := testcomponents.NewTestRenderer(counter2)

	// Act: Render both
	vnode1 := renderer1.RenderRoot()
	vnode2 := renderer2.RenderRoot()

	// Assert: Each component has its own state
	if vnode1.Children[0].Content != "Count: 10" {
		t.Errorf("Counter1 count incorrect: %s", vnode1.Children[0].Content)
	}
	if vnode2.Children[0].Content != "Count: 20" {
		t.Errorf("Counter2 count incorrect: %s", vnode2.Children[0].Content)
	}

	// Act: Update only counter1
	counter1.Increment()

	// Assert: Only counter1's VDOM changed
	vnode1 = renderer1.GetCurrentVDOM()
	vnode2 = renderer2.GetCurrentVDOM()

	if vnode1.Children[0].Content != "Count: 11" {
		t.Errorf("Counter1 should be 11, got: %s", vnode1.Children[0].Content)
	}
	if vnode2.Children[0].Content != "Count: 20" {
		t.Errorf("Counter2 should still be 20, got: %s", vnode2.Children[0].Content)
	}
}
