package multiline

import (
	"strings"
	"testing"

	"github.com/ForgeLogic/nojs-compiler/testcomponents"
	"github.com/ForgeLogic/nojs/vdom"
)

// TestSingleLineBasicRendering tests that single-line HTML tags with data bindings compile and render correctly.
func TestSingleLineBasicRendering(t *testing.T) {
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

	// Check first h1 (single-line)
	h1Single := vnode.Children[0]
	if h1Single.Tag != "h1" {
		t.Errorf("Expected first child to be 'h1', got '%s'", h1Single.Tag)
	}
	if h1Single.Content != "Single-line: Test Title" {
		t.Errorf("Expected 'Single-line: Test Title', got '%s'", h1Single.Content)
	}

	// Check first p (single-line)
	pSingle := vnode.Children[2]
	if pSingle.Tag != "p" {
		t.Errorf("Expected third child to be 'p', got '%s'", pSingle.Tag)
	}
	if pSingle.Content != "Single-line paragraph: Test Message" {
		t.Errorf("Expected 'Single-line paragraph: Test Message', got '%s'", pSingle.Content)
	}

	// Check h2 (single-line)
	h2 := vnode.Children[4]
	if h2.Tag != "h2" {
		t.Errorf("Expected fifth child to be 'h2', got '%s'", h2.Tag)
	}
	if h2.Content != "42" {
		t.Errorf("Expected '42', got '%s'", h2.Content)
	}
}

// TestSingleLineDataBinding verifies that data bindings work in single-line format with various edge cases.
func TestSingleLineDataBinding(t *testing.T) {
	testCases := []struct {
		name    string
		title   string
		message string
		count   int
		checkFn func(t *testing.T, vnode *vdom.VNode)
	}{
		{
			name:    "Empty strings",
			title:   "",
			message: "",
			count:   0,
			checkFn: func(t *testing.T, vnode *vdom.VNode) {
				// First h1 should have "Single-line: " (no title)
				h1Content := vnode.Children[0].Content
				if !strings.Contains(h1Content, "Single-line: ") {
					t.Errorf("Expected 'Single-line: ' in first h1, got: %s", h1Content)
				}
				// Should not have extra whitespace
				if strings.HasPrefix(h1Content, "\n") || strings.HasSuffix(h1Content, "\n") {
					t.Errorf("Single-line h1 should not have leading/trailing newlines, got: %q", h1Content)
				}
			},
		},
		{
			name:    "Special characters",
			title:   "Test <Title> & \"Quotes\"",
			message: "Message with\ttabs",
			count:   -999,
			checkFn: func(t *testing.T, vnode *vdom.VNode) {
				// Verify special characters are preserved in single-line format
				h1Text := vnode.Children[0].Content
				if !strings.Contains(h1Text, "<Title>") || !strings.Contains(h1Text, "\"Quotes\"") {
					t.Errorf("Special characters not preserved in single-line h1: %s", h1Text)
				}

				// Verify tabs are preserved in single-line format
				pText := vnode.Children[2].Content
				if !strings.Contains(pText, "\t") {
					t.Errorf("Tab character not preserved in single-line p: %s", pText)
				}

				// Verify negative number in single-line format
				h2Text := vnode.Children[4].Content
				if h2Text != "-999" {
					t.Errorf("Expected '-999', got '%s'", h2Text)
				}
			},
		},
		{
			name:    "Very long text",
			title:   strings.Repeat("A", 1000),
			message: strings.Repeat("B", 1000),
			count:   999999,
			checkFn: func(t *testing.T, vnode *vdom.VNode) {
				// Verify long strings are handled correctly in single-line format
				h1Content := vnode.Children[0].Content
				if len(h1Content) < 1000 {
					t.Errorf("Long title not fully rendered in single-line h1")
				}
				// Should not have extra whitespace even with long text
				if strings.HasPrefix(h1Content, "\n") || strings.HasSuffix(h1Content, "\n") {
					t.Errorf("Single-line h1 should not have newlines even with long text")
				}

				pContent := vnode.Children[2].Content
				if len(pContent) < 1000 {
					t.Errorf("Long message not fully rendered in single-line p")
				}
				// Should not have extra whitespace even with long text
				if strings.HasPrefix(pContent, "\n") || strings.HasSuffix(pContent, "\n") {
					t.Errorf("Single-line p should not have newlines even with long text")
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

// TestSingleLineNoExtraWhitespace verifies that single-line tags don't have unexpected whitespace.
func TestSingleLineNoExtraWhitespace(t *testing.T) {
	comp := &MultilineText{
		Title:   "Title",
		Message: "Message",
		Count:   1,
	}

	renderer := testcomponents.NewTestRenderer(comp)
	vnode := renderer.RenderRoot()

	// Single-line h1 should NOT have leading/trailing whitespace
	h1Single := vnode.Children[0]
	if strings.HasPrefix(h1Single.Content, "\n") || strings.HasSuffix(h1Single.Content, "\n") {
		t.Errorf("Single-line h1 should not have leading/trailing newlines, got: %q", h1Single.Content)
	}
	if strings.HasPrefix(h1Single.Content, " ") && strings.Count(h1Single.Content, " ") > strings.Count("Single-line: Title", " ") {
		t.Errorf("Single-line h1 has unexpected leading spaces, got: %q", h1Single.Content)
	}

	// Single-line p should NOT have leading/trailing whitespace
	pSingle := vnode.Children[2]
	if strings.HasPrefix(pSingle.Content, "\n") || strings.HasSuffix(pSingle.Content, "\n") {
		t.Errorf("Single-line p should not have leading/trailing newlines, got: %q", pSingle.Content)
	}
	if strings.HasPrefix(pSingle.Content, " ") && strings.Count(pSingle.Content, " ") > strings.Count("Single-line paragraph: Message", " ") {
		t.Errorf("Single-line p has unexpected leading spaces, got: %q", pSingle.Content)
	}

	// Single-line h2 should be just the number
	h2Single := vnode.Children[4]
	if h2Single.Content != "1" {
		t.Errorf("Expected h2 content to be '1', got: %q", h2Single.Content)
	}
}
