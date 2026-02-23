package multiline

import (
	"strings"
	"testing"

	"github.com/ForgeLogic/nojs-compiler/testcomponents"
	"github.com/ForgeLogic/nojs/vdom"
)

// TestMultiLineBasicRendering tests that multi-line HTML tags with data bindings compile and render correctly.
func TestMultiLineBasicRendering(t *testing.T) {
	// Create the component with test data
	comp := &MultilineText{
		Title:   "Test Title",
		Message: "Test Message",
		Count:   42,
	}

	// Create a test renderer and perform initial render
	renderer := testcomponents.NewTestRenderer(comp)
	vnode := renderer.RenderRoot()

	// Verify the root element is a div
	if vnode.Tag != "div" {
		t.Fatalf("Expected root element to be 'div', got '%s'", vnode.Tag)
	}

	// We should have 6 children (2 h1s, 2 ps, 1 h2, 1 h3)
	if len(vnode.Children) != 6 {
		t.Fatalf("Expected 6 children, got %d", len(vnode.Children))
	}

	// Check second h1 (multi-line) - should preserve whitespace
	h1Multi := vnode.Children[1]
	if h1Multi.Tag != "h1" {
		t.Errorf("Expected second child to be 'h1', got '%s'", h1Multi.Tag)
	}
	expectedH1Multi := "\n        Multi-line: Test Title\n    "
	if h1Multi.Content != expectedH1Multi {
		t.Errorf("Expected multi-line h1 text with whitespace:\n%q\ngot:\n%q", expectedH1Multi, h1Multi.Content)
	}

	// Check second p (multi-line) - should preserve whitespace
	pMulti := vnode.Children[3]
	if pMulti.Tag != "p" {
		t.Errorf("Expected fourth child to be 'p', got '%s'", pMulti.Tag)
	}
	expectedPMulti := "\n        Multi-line paragraph: Test Message\n    "
	if pMulti.Content != expectedPMulti {
		t.Errorf("Expected multi-line p text with whitespace:\n%q\ngot:\n%q", expectedPMulti, pMulti.Content)
	}

	// Check h3 (multi-line)
	h3 := vnode.Children[5]
	if h3.Tag != "h3" {
		t.Errorf("Expected sixth child to be 'h3', got '%s'", h3.Tag)
	}
	expectedH3 := "\n        42\n    "
	if h3.Content != expectedH3 {
		t.Errorf("Expected multi-line h3 text with whitespace:\n%q\ngot:\n%q", expectedH3, h3.Content)
	}
}

// TestMultiLineDataBinding verifies that data bindings work in multi-line format with various edge cases.
func TestMultiLineDataBinding(t *testing.T) {
	testCases := []struct {
		name    string
		title   string
		message string
		count   int
		checkFn func(t *testing.T, vnode *vdom.VNode)
	}{
		{
			name:    "Empty strings with whitespace preservation",
			title:   "",
			message: "",
			count:   0,
			checkFn: func(t *testing.T, vnode *vdom.VNode) {
				// Second h1 (multi-line) should have "Multi-line: " with whitespace preserved
				h1Content := vnode.Children[1].Content
				if !strings.Contains(h1Content, "Multi-line: ") {
					t.Errorf("Expected 'Multi-line: ' in second h1, got: %s", h1Content)
				}
				// Should have whitespace even with empty title
				if !strings.HasPrefix(h1Content, "\n        ") {
					t.Errorf("Multi-line h1 should have leading whitespace, got: %q", h1Content)
				}
				if !strings.HasSuffix(h1Content, "\n    ") {
					t.Errorf("Multi-line h1 should have trailing whitespace, got: %q", h1Content)
				}
			},
		},
		{
			name:    "Special characters with whitespace",
			title:   "Test <Title> & \"Quotes\"",
			message: "Message with\ttabs",
			count:   -999,
			checkFn: func(t *testing.T, vnode *vdom.VNode) {
				// Verify special characters are preserved in multi-line format with whitespace
				h1Text := vnode.Children[1].Content
				if !strings.Contains(h1Text, "<Title>") || !strings.Contains(h1Text, "\"Quotes\"") {
					t.Errorf("Special characters not preserved in multi-line h1: %s", h1Text)
				}
				// Should still have leading/trailing whitespace
				if !strings.HasPrefix(h1Text, "\n        ") {
					t.Errorf("Multi-line h1 should have leading whitespace, got: %q", h1Text)
				}
				if !strings.HasSuffix(h1Text, "\n    ") {
					t.Errorf("Multi-line h1 should have trailing whitespace, got: %q", h1Text)
				}

				// Verify tabs are preserved in multi-line format with whitespace
				pText := vnode.Children[3].Content
				if !strings.Contains(pText, "\t") {
					t.Errorf("Tab character not preserved in multi-line p: %s", pText)
				}
				// Should still have leading/trailing whitespace
				if !strings.HasPrefix(pText, "\n        ") {
					t.Errorf("Multi-line p should have leading whitespace, got: %q", pText)
				}
				if !strings.HasSuffix(pText, "\n    ") {
					t.Errorf("Multi-line p should have trailing whitespace, got: %q", pText)
				}
			},
		},
		{
			name:    "Very long text with whitespace preservation",
			title:   strings.Repeat("A", 1000),
			message: strings.Repeat("B", 1000),
			count:   999999,
			checkFn: func(t *testing.T, vnode *vdom.VNode) {
				// Verify long strings are handled correctly in multi-line format
				h1Content := vnode.Children[1].Content
				if len(h1Content) < 1000 {
					t.Errorf("Long title not fully rendered in multi-line h1")
				}
				// Should preserve whitespace even with long text
				if !strings.HasPrefix(h1Content, "\n        ") {
					t.Errorf("Multi-line h1 should have leading whitespace even with long text")
				}
				if !strings.HasSuffix(h1Content, "\n    ") {
					t.Errorf("Multi-line h1 should have trailing whitespace even with long text")
				}

				pContent := vnode.Children[3].Content
				if len(pContent) < 1000 {
					t.Errorf("Long message not fully rendered in multi-line p")
				}
				// Should preserve whitespace even with long text
				if !strings.HasPrefix(pContent, "\n        ") {
					t.Errorf("Multi-line p should have leading whitespace even with long text")
				}
				if !strings.HasSuffix(pContent, "\n    ") {
					t.Errorf("Multi-line p should have trailing whitespace even with long text")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			comp := &MultilineText{
				Title:   tc.title,
				Message: tc.message,
				Count:   tc.count,
			}

			renderer := testcomponents.NewTestRenderer(comp)
			vnode := renderer.RenderRoot()
			tc.checkFn(t, vnode)
		})
	}
}

// TestMultiLineWhitespacePreservation specifically tests that whitespace is correctly preserved in multi-line tags.
func TestMultiLineWhitespacePreservation(t *testing.T) {
	comp := &MultilineText{
		Title:   "Title",
		Message: "Message",
		Count:   1,
	}

	renderer := testcomponents.NewTestRenderer(comp)
	vnode := renderer.RenderRoot()

	// Multi-line h1 should preserve leading/trailing whitespace
	h1Multi := vnode.Children[1]
	if !strings.HasPrefix(h1Multi.Content, "\n        ") {
		t.Errorf("Multi-line h1 should preserve leading whitespace, got: %q", h1Multi.Content)
	}
	if !strings.HasSuffix(h1Multi.Content, "\n    ") {
		t.Errorf("Multi-line h1 should preserve trailing whitespace, got: %q", h1Multi.Content)
	}

	// Multi-line p should preserve leading/trailing whitespace
	pMulti := vnode.Children[3]
	if !strings.HasPrefix(pMulti.Content, "\n        ") {
		t.Errorf("Multi-line p should preserve leading whitespace, got: %q", pMulti.Content)
	}
	if !strings.HasSuffix(pMulti.Content, "\n    ") {
		t.Errorf("Multi-line p should preserve trailing whitespace, got: %q", pMulti.Content)
	}

	// Multi-line h3 should preserve leading/trailing whitespace
	h3Multi := vnode.Children[5]
	if !strings.HasPrefix(h3Multi.Content, "\n        ") {
		t.Errorf("Multi-line h3 should preserve leading whitespace, got: %q", h3Multi.Content)
	}
	if !strings.HasSuffix(h3Multi.Content, "\n    ") {
		t.Errorf("Multi-line h3 should preserve trailing whitespace, got: %q", h3Multi.Content)
	}
}

// TestMultiLineExactWhitespacePattern verifies the exact whitespace pattern in multi-line tags.
func TestMultiLineExactWhitespacePattern(t *testing.T) {
	comp := &MultilineText{
		Title:   "TestTitle",
		Message: "TestMessage",
		Count:   123,
	}

	renderer := testcomponents.NewTestRenderer(comp)
	vnode := renderer.RenderRoot()

	// Test exact pattern for multi-line h1
	h1Multi := vnode.Children[1]
	expectedH1 := "\n        Multi-line: TestTitle\n    "
	if h1Multi.Content != expectedH1 {
		t.Errorf("Multi-line h1 has incorrect whitespace pattern.\nExpected: %q\nGot:      %q", expectedH1, h1Multi.Content)
	}

	// Test exact pattern for multi-line p
	pMulti := vnode.Children[3]
	expectedP := "\n        Multi-line paragraph: TestMessage\n    "
	if pMulti.Content != expectedP {
		t.Errorf("Multi-line p has incorrect whitespace pattern.\nExpected: %q\nGot:      %q", expectedP, pMulti.Content)
	}

	// Test exact pattern for multi-line h3
	h3Multi := vnode.Children[5]
	expectedH3 := "\n        123\n    "
	if h3Multi.Content != expectedH3 {
		t.Errorf("Multi-line h3 has incorrect whitespace pattern.\nExpected: %q\nGot:      %q", expectedH3, h3Multi.Content)
	}
}
