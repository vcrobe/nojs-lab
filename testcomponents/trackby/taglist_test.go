//go:build !wasm
// +build !wasm

package trackby

import (
	"testing"

	"github.com/vcrobe/nojs/testcomponents"
	"github.com/vcrobe/nojs/vdom"
)

// TestTagList_BareVariableTrackBy_InitialRender verifies that trackBy with a bare variable
// (primitive string) correctly renders the initial list of tags.
func TestTagList_BareVariableTrackBy_InitialRender(t *testing.T) {
	// Arrange: Create a TagList component with initial tags
	tagList := &TagList{
		Tags: []string{"golang", "wasm", "component", "framework"},
	}
	renderer := testcomponents.NewTestRenderer(tagList)

	// Act: Perform initial render
	vnode := renderer.RenderRoot()

	// Assert: Verify root structure
	if vnode.Tag != "div" {
		t.Errorf("Expected root tag 'div', got '%s'", vnode.Tag)
	}

	// Assert: Find the <ul> element
	var ulNode *vdom.VNode
	for _, child := range vnode.Children {
		if child.Tag == "ul" {
			ulNode = child
			break
		}
	}
	if ulNode == nil {
		t.Fatalf("Expected <ul> element not found in root children")
	}

	// Assert: Verify the number of list items matches the tags
	expectedTagCount := 4 // "golang", "wasm", "component", "framework"
	if len(ulNode.Children) != expectedTagCount {
		t.Fatalf("Expected %d <li> elements, got %d", expectedTagCount, len(ulNode.Children))
	}

	// Assert: Verify each tag is rendered correctly
	expectedTags := []string{"golang", "wasm", "component", "framework"}
	for i, expectedTag := range expectedTags {
		liNode := ulNode.Children[i]
		if liNode.Tag != "li" {
			t.Errorf("Expected child %d to be 'li', got '%s'", i, liNode.Tag)
		}
		expectedContent := "Tag " + string(rune('0'+i)) + ": " + expectedTag
		if liNode.Content != expectedContent {
			t.Errorf("Child %d: expected '%s', got '%s'", i, expectedContent, liNode.Content)
		}
	}
}

// TestTagList_BareVariableTrackBy_AddTag verifies that calling AddTag() triggers a re-render
// with the new tag added to the list.
func TestTagList_BareVariableTrackBy_AddTag(t *testing.T) {
	// Arrange
	tagList := &TagList{
		Tags: []string{"golang", "wasm", "component", "framework"},
	}
	renderer := testcomponents.NewTestRenderer(tagList)
	vnode1 := renderer.RenderRoot()

	// Find the initial <ul> element
	var ulNode1 *vdom.VNode
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}
	if len(ulNode1.Children) != 4 {
		t.Fatalf("Initial tag count should be 4, got %d", len(ulNode1.Children))
	}

	// Act: Add a new tag
	tagList.AddTag("testing")

	// Assert: Verify the VDOM was updated
	vnode2 := renderer.GetCurrentVDOM()
	var ulNode2 *vdom.VNode
	for _, child := range vnode2.Children {
		if child.Tag == "ul" {
			ulNode2 = child
			break
		}
	}

	expectedTagCount := 5
	if len(ulNode2.Children) != expectedTagCount {
		t.Fatalf("After adding a tag, expected %d <li> elements, got %d", expectedTagCount, len(ulNode2.Children))
	}

	// Assert: Verify the new tag was added at the end
	lastLiNode := ulNode2.Children[4]
	expectedContent := "Tag 4: testing"
	if lastLiNode.Content != expectedContent {
		t.Errorf("Expected last tag '%s', got '%s'", expectedContent, lastLiNode.Content)
	}
}

// TestTagList_BareVariableTrackBy_MultipleAdditions verifies that multiple AddTag() calls
// each trigger re-renders with cumulative additions.
func TestTagList_BareVariableTrackBy_MultipleAdditions(t *testing.T) {
	// Arrange
	tagList := &TagList{
		Tags: []string{"golang", "wasm", "component", "framework"},
	}
	renderer := testcomponents.NewTestRenderer(tagList)
	renderer.RenderRoot()

	// Act & Assert: Add multiple tags
	tagsToAdd := []string{"rust", "typescript", "python"}
	for i, newTag := range tagsToAdd {
		tagList.AddTag(newTag)
		vnode := renderer.GetCurrentVDOM()

		var ulNode *vdom.VNode
		for _, child := range vnode.Children {
			if child.Tag == "ul" {
				ulNode = child
				break
			}
		}

		expectedCount := 4 + i + 1
		if len(ulNode.Children) != expectedCount {
			t.Errorf("After adding tag %d, expected %d items, got %d", i+1, expectedCount, len(ulNode.Children))
		}

		// Verify the newly added tag
		lastLiNode := ulNode.Children[len(ulNode.Children)-1]
		expectedContent := "Tag " + string(rune('0'+expectedCount-1)) + ": " + newTag
		if lastLiNode.Content != expectedContent {
			t.Errorf("Tag %d: expected '%s', got '%s'", i+1, expectedContent, lastLiNode.Content)
		}
	}
}

// TestTagList_BareVariableTrackBy_ClearTags verifies that ClearTags() removes all tags
// and triggers a re-render with an empty list.
func TestTagList_BareVariableTrackBy_ClearTags(t *testing.T) {
	// Arrange
	tagList := &TagList{
		Tags: []string{"golang", "wasm", "component", "framework"},
	}
	renderer := testcomponents.NewTestRenderer(tagList)
	vnode1 := renderer.RenderRoot()

	// Verify initial state
	var ulNode1 *vdom.VNode
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}
	if len(ulNode1.Children) != 4 {
		t.Fatalf("Initial tag count should be 4, got %d", len(ulNode1.Children))
	}

	// Act: Clear all tags
	tagList.ClearTags()

	// Assert: Verify the VDOM was updated with an empty list
	vnode2 := renderer.GetCurrentVDOM()
	var ulNode2 *vdom.VNode
	for _, child := range vnode2.Children {
		if child.Tag == "ul" {
			ulNode2 = child
			break
		}
	}

	if len(ulNode2.Children) != 0 {
		t.Errorf("After clearing tags, expected 0 items, got %d", len(ulNode2.Children))
	}
}

// TestTagList_BareVariableTrackBy_AddAfterClear verifies that AddTag() works correctly
// after clearing the tag list.
func TestTagList_BareVariableTrackBy_AddAfterClear(t *testing.T) {
	// Arrange
	tagList := &TagList{
		Tags: []string{"golang", "wasm", "component", "framework"},
	}
	renderer := testcomponents.NewTestRenderer(tagList)
	renderer.RenderRoot()

	// Act: Clear and then add new tags
	tagList.ClearTags()
	tagList.AddTag("newonly")

	// Assert: Verify the list contains only the newly added tag
	vnode := renderer.GetCurrentVDOM()
	var ulNode *vdom.VNode
	for _, child := range vnode.Children {
		if child.Tag == "ul" {
			ulNode = child
			break
		}
	}

	if len(ulNode.Children) != 1 {
		t.Fatalf("After clear and add, expected 1 item, got %d", len(ulNode.Children))
	}

	liNode := ulNode.Children[0]
	expectedContent := "Tag 0: newonly"
	if liNode.Content != expectedContent {
		t.Errorf("Expected '%s', got '%s'", expectedContent, liNode.Content)
	}
}

// TestTagList_BareVariableTrackBy_RenderIsolation verifies that multiple TagList instances
// maintain separate state and VDOM.
func TestTagList_BareVariableTrackBy_RenderIsolation(t *testing.T) {
	// Arrange: Create two separate TagList instances
	tagList1 := &TagList{
		Tags: []string{"golang", "wasm", "component", "framework"},
	}
	tagList2 := &TagList{
		Tags: []string{"golang", "wasm", "component", "framework"},
	}

	renderer1 := testcomponents.NewTestRenderer(tagList1)
	renderer2 := testcomponents.NewTestRenderer(tagList2)

	// Act: Render both
	vnode1 := renderer1.RenderRoot()
	vnode2 := renderer2.RenderRoot()

	// Assert: Find ul nodes
	var ulNode1, ulNode2 *vdom.VNode
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}
	for _, child := range vnode2.Children {
		if child.Tag == "ul" {
			ulNode2 = child
			break
		}
	}

	// Both should have the same initial tags (4)
	if len(ulNode1.Children) != 4 || len(ulNode2.Children) != 4 {
		t.Fatalf("Both instances should have 4 initial tags")
	}

	// Act: Modify only the first instance
	tagList1.AddTag("unique1")

	// Assert: Only the first instance was modified
	vnode1 = renderer1.GetCurrentVDOM()
	vnode2 = renderer2.GetCurrentVDOM()

	ulNode1 = nil
	ulNode2 = nil
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}
	for _, child := range vnode2.Children {
		if child.Tag == "ul" {
			ulNode2 = child
			break
		}
	}

	if len(ulNode1.Children) != 5 {
		t.Errorf("First instance should have 5 tags after add, got %d", len(ulNode1.Children))
	}
	if len(ulNode2.Children) != 4 {
		t.Errorf("Second instance should still have 4 tags, got %d", len(ulNode2.Children))
	}
}
