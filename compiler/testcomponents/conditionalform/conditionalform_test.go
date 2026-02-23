//go:build !wasm

package conditionalform

import (
	"testing"

	"github.com/ForgeLogic/nojs-compiler/testcomponents"
)

// previewDiv extracts the conditional div (live-preview or live-preview muted)
// from the rendered root, failing the test if the structure is unexpected.
func previewDiv(t *testing.T, renderer *testcomponents.TestRenderer) string {
	t.Helper()
	root := renderer.GetCurrentVDOM()
	if root == nil {
		t.Fatal("GetCurrentVDOM returned nil")
	}
	if root.Tag != "div" {
		t.Fatalf("Expected root tag 'div', got '%s'", root.Tag)
	}
	if len(root.Children) == 0 {
		t.Fatal("Root div has no children")
	}
	preview := root.Children[0]
	if preview.Tag != "div" {
		t.Fatalf("Expected preview child tag 'div', got '%s'", preview.Tag)
	}
	if preview.Attributes == nil {
		t.Fatal("Preview div has no attributes")
	}
	classVal, _ := preview.Attributes["class"].(string)
	return classVal
}

// TestConditionalForm_InitialRender_ShowsMutedPlaceholder verifies that on the
// initial render (no name entered) the muted placeholder branch is rendered.
func TestConditionalForm_InitialRender_ShowsMutedPlaceholder(t *testing.T) {
	comp := &ConditionalForm{}
	renderer := testcomponents.NewTestRenderer(comp)
	renderer.RenderRoot()

	class := previewDiv(t, renderer)
	if class != "live-preview muted" {
		t.Errorf("Expected class 'live-preview muted', got '%s'", class)
	}
}

// TestConditionalForm_TypeName_ShowsLivePreview verifies that after the user
// types a name the live-preview branch is rendered with the correct content.
func TestConditionalForm_TypeName_ShowsLivePreview(t *testing.T) {
	comp := &ConditionalForm{}
	renderer := testcomponents.NewTestRenderer(comp)
	renderer.RenderRoot()

	comp.SetName("Alice")

	class := previewDiv(t, renderer)
	if class != "live-preview" {
		t.Errorf("Expected class 'live-preview', got '%s'", class)
	}

	root := renderer.GetCurrentVDOM()
	preview := root.Children[0]
	if len(preview.Children) == 0 {
		t.Fatal("Live-preview div has no children")
	}
	p := preview.Children[0]
	if p.Tag != "p" {
		t.Fatalf("Expected 'p' inside live-preview, got '%s'", p.Tag)
	}
	// The p contains: Text("Hello, "), span{Alice}, Text("!")
	if len(p.Children) != 3 {
		t.Fatalf("Expected 3 children in p (text, span, text), got %d", len(p.Children))
	}
	spanNode := p.Children[1]
	if len(spanNode.Children) == 0 {
		t.Fatal("span has no children")
	}
	if spanNode.Children[0].Content != "Alice" {
		t.Errorf("Expected span text 'Alice', got '%s'", spanNode.Children[0].Content)
	}
}

// TestConditionalForm_TypeThenClear_RestoresMutedPlaceholder is the regression
// test for the bug where clearing the input after typing a name failed to
// restore the muted placeholder branch in the DOM.
//
// Repro steps (browser):
//  1. Load the Forms & Events page.
//  2. Type any name -> the Live Preview block appears.
//  3. Clear the input -> the "Start typing..." placeholder should reappear.
//
// Before the fix, step 3 left the DOM blank because patchElement set
// textContent on the <p> in the muted branch, but the subsequent
// patchChildren call removed that text node.
func TestConditionalForm_TypeThenClear_RestoresMutedPlaceholder(t *testing.T) {
	comp := &ConditionalForm{}
	renderer := testcomponents.NewTestRenderer(comp)
	renderer.RenderRoot()

	// Step 1: type a name
	comp.SetName("Alice")
	class := previewDiv(t, renderer)
	if class != "live-preview" {
		t.Errorf("After typing: expected class 'live-preview', got '%s'", class)
	}

	// Step 2: clear the input
	comp.SetName("")
	class = previewDiv(t, renderer)
	if class != "live-preview muted" {
		t.Errorf("After clearing: expected class 'live-preview muted', got '%s'", class)
	}

	// The placeholder paragraph must be present with the expected text
	root := renderer.GetCurrentVDOM()
	preview := root.Children[0]
	if len(preview.Children) == 0 {
		t.Fatal("Muted placeholder div has no children after clearing input")
	}
	p := preview.Children[0]
	if p.Tag != "p" {
		t.Fatalf("Expected 'p' inside muted placeholder div, got '%s'", p.Tag)
	}
	const expectedText = "Start typing your name above to see the live preview..."
	if p.Content != expectedText {
		t.Errorf("Expected placeholder text\n  %q\ngot\n  %q", expectedText, p.Content)
	}
}

// TestConditionalForm_MultipleTypeClearCycles verifies that multiple
// type-then-clear cycles continue to toggle the conditional correctly.
func TestConditionalForm_MultipleTypeClearCycles(t *testing.T) {
	comp := &ConditionalForm{}
	renderer := testcomponents.NewTestRenderer(comp)
	renderer.RenderRoot()

	for i, name := range []string{"Alice", "", "Bob", "", "Charlie", ""} {
		comp.SetName(name)
		class := previewDiv(t, renderer)

		wantMuted := name == ""
		if wantMuted && class != "live-preview muted" {
			t.Errorf("cycle %d (clear): expected 'live-preview muted', got '%s'", i, class)
		}
		if !wantMuted && class != "live-preview" {
			t.Errorf("cycle %d (name=%q): expected 'live-preview', got '%s'", i, name, class)
		}
	}
}
