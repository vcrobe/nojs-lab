//go:build !wasm
// +build !wasm

package trackby

import (
	"testing"

	"github.com/ForgeLogic/nojs-compiler/testcomponents"
	"github.com/ForgeLogic/nojs/vdom"
)

// TestMultiItemList_MultipleChildrenPerIteration_InitialRender verifies that a loop with multiple
// sibling child elements per iteration renders correctly without variable shadowing errors.
// This tests the critical compiler bug fix where multiple children in a loop caused duplicate
// variable declaration errors.
func TestMultiItemList_MultipleChildrenPerIteration_InitialRender(t *testing.T) {
	// Arrange: Create a MultiItemList component with initial items
	multiList := &MultiItemList{
		Items: []Item{
			{ID: 101, Name: "Alpha"},
			{ID: 102, Name: "Beta"},
			{ID: 103, Name: "Gamma"},
		},
	}
	renderer := testcomponents.NewTestRenderer(multiList)

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

	// Assert: Verify the number of list items - 3 items × 2 elements per item = 6 total <li> elements
	expectedLiCount := 6 // 3 items with 2 <li> siblings each
	if len(ulNode.Children) != expectedLiCount {
		t.Fatalf("Expected %d <li> elements (3 items × 2 elements), got %d", expectedLiCount, len(ulNode.Children))
	}

	// Assert: Verify the structure of rendered items
	// Expected pattern: [item_0_li_name, item_0_li_metadata, item_1_li_name, item_1_li_metadata, item_2_li_name, item_2_li_metadata]
	expectedItems := []struct {
		id   int
		name string
	}{
		{101, "Alpha"},
		{102, "Beta"},
		{103, "Gamma"},
	}

	for itemIdx, expectedItem := range expectedItems {
		baseIdx := itemIdx * 2

		// Check first <li> (name element)
		nameLi := ulNode.Children[baseIdx]
		if nameLi.Tag != "li" {
			t.Errorf("Item %d (name): expected 'li', got '%s'", itemIdx, nameLi.Tag)
		}

		// Check second <li> (metadata element with class)
		metadataLi := ulNode.Children[baseIdx+1]
		if metadataLi.Tag != "li" {
			t.Errorf("Item %d (metadata): expected 'li', got '%s'", itemIdx, metadataLi.Tag)
		}

		// Verify metadata contains the item ID
		if !contains(metadataLi.Content, string(rune('0'+expectedItem.id%10))) {
			t.Errorf("Item %d metadata: expected to contain ID %d, got '%s'", itemIdx, expectedItem.id, metadataLi.Content)
		}
	}
}

// TestMultiItemList_MultipleChildrenPerIteration_AddItem verifies that adding items and re-rendering
// correctly generates multiple children per iteration.
func TestMultiItemList_MultipleChildrenPerIteration_AddItem(t *testing.T) {
	// Arrange
	multiList := &MultiItemList{
		Items: []Item{
			{ID: 101, Name: "Alpha"},
			{ID: 102, Name: "Beta"},
			{ID: 103, Name: "Gamma"},
		},
	}
	renderer := testcomponents.NewTestRenderer(multiList)
	vnode1 := renderer.RenderRoot()

	// Find the <ul> and verify initial state
	var ulNode1 *vdom.VNode
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}
	if len(ulNode1.Children) != 6 {
		t.Fatalf("Initial <li> count should be 6, got %d", len(ulNode1.Children))
	}

	// Act: Add a new item
	multiList.AddItem("Delta")

	// Assert: Verify the VDOM was updated
	vnode2 := renderer.GetCurrentVDOM()
	var ulNode2 *vdom.VNode
	for _, child := range vnode2.Children {
		if child.Tag == "ul" {
			ulNode2 = child
			break
		}
	}

	// After adding 1 item, we should have 4 items × 2 elements = 8 <li> elements
	expectedLiCount := 8
	if len(ulNode2.Children) != expectedLiCount {
		t.Fatalf("After adding item, expected %d <li> elements, got %d", expectedLiCount, len(ulNode2.Children))
	}

	// Verify the new item's metadata is at the end
	lastMetadataLi := ulNode2.Children[7]
	if lastMetadataLi.Tag != "li" {
		t.Errorf("Last element: expected 'li', got '%s'", lastMetadataLi.Tag)
	}

	// The new item should have ID 104
	if !contains(lastMetadataLi.Content, "104") {
		t.Errorf("Expected new item ID 104 in metadata, got '%s'", lastMetadataLi.Content)
	}
}

// TestMultiItemList_MultipleChildrenPerIteration_ClearItems verifies that clearing items
// removes all rendered children correctly.
func TestMultiItemList_MultipleChildrenPerIteration_ClearItems(t *testing.T) {
	// Arrange
	multiList := &MultiItemList{
		Items: []Item{
			{ID: 101, Name: "Alpha"},
			{ID: 102, Name: "Beta"},
			{ID: 103, Name: "Gamma"},
		},
	}
	renderer := testcomponents.NewTestRenderer(multiList)
	vnode1 := renderer.RenderRoot()

	// Verify initial state
	var ulNode1 *vdom.VNode
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}
	if len(ulNode1.Children) != 6 {
		t.Fatalf("Initial <li> count should be 6, got %d", len(ulNode1.Children))
	}

	// Act: Clear all items
	multiList.ClearItems()

	// Assert: Verify the VDOM was updated with empty list
	vnode2 := renderer.GetCurrentVDOM()
	var ulNode2 *vdom.VNode
	for _, child := range vnode2.Children {
		if child.Tag == "ul" {
			ulNode2 = child
			break
		}
	}

	if len(ulNode2.Children) != 0 {
		t.Errorf("After clearing items, expected 0 <li> elements, got %d", len(ulNode2.Children))
	}
}

// TestMultiItemList_MultipleChildrenPerIteration_RenderIsolation verifies that multiple instances
// maintain separate state and VDOM.
func TestMultiItemList_MultipleChildrenPerIteration_RenderIsolation(t *testing.T) {
	// Arrange: Create two separate instances
	multiList1 := &MultiItemList{
		Items: []Item{
			{ID: 101, Name: "Alpha"},
			{ID: 102, Name: "Beta"},
			{ID: 103, Name: "Gamma"},
		},
	}
	multiList2 := &MultiItemList{
		Items: []Item{
			{ID: 201, Name: "One"},
			{ID: 202, Name: "Two"},
		},
	}

	renderer1 := testcomponents.NewTestRenderer(multiList1)
	renderer2 := testcomponents.NewTestRenderer(multiList2)

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

	// First instance: 3 items × 2 elements = 6 <li>
	// Second instance: 2 items × 2 elements = 4 <li>
	if len(ulNode1.Children) != 6 {
		t.Errorf("Instance 1 should have 6 <li> elements, got %d", len(ulNode1.Children))
	}
	if len(ulNode2.Children) != 4 {
		t.Errorf("Instance 2 should have 4 <li> elements, got %d", len(ulNode2.Children))
	}

	// Act: Modify only the first instance
	multiList1.AddItem("Delta")

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

	// After add, instance 1 should have 8 <li> elements (4 items × 2)
	if len(ulNode1.Children) != 8 {
		t.Errorf("Instance 1 after add should have 8 <li> elements, got %d", len(ulNode1.Children))
	}
	// Instance 2 should still have 4 <li> elements
	if len(ulNode2.Children) != 4 {
		t.Errorf("Instance 2 should still have 4 <li> elements, got %d", len(ulNode2.Children))
	}
}

// TestMultiItemList_MultipleChildrenPerIteration_VariableNamesUnique verifies that the compiler
// generates unique variable names for each child element in the loop, preventing variable shadowing.
// This is a regression test for the bug where multiple children used the same variable name.
func TestMultiItemList_MultipleChildrenPerIteration_VariableNamesUnique(t *testing.T) {
	// Arrange: This test primarily validates that compilation succeeds.
	// If the compiler generated duplicate variable names like before, this would fail at compile time.
	multiList := &MultiItemList{
		Items: []Item{
			{ID: 101, Name: "Alpha"},
			{ID: 102, Name: "Beta"},
		},
	}
	renderer := testcomponents.NewTestRenderer(multiList)

	// Act: Render the component multiple times to ensure no state issues
	vnode := renderer.RenderRoot()
	for range 5 {
		multiList.AddItem("Item")
		vnode = renderer.GetCurrentVDOM()
	}

	// Assert: Verify final render is valid
	var ulNode *vdom.VNode
	for _, child := range vnode.Children {
		if child.Tag == "ul" {
			ulNode = child
			break
		}
	}

	// Started with 2 items, added 5 more = 7 items × 2 elements = 14 <li> elements
	expectedCount := 14
	if len(ulNode.Children) != expectedCount {
		t.Errorf("Expected %d <li> elements, got %d", expectedCount, len(ulNode.Children))
	}

	t.Log("✓ Compiler generated unique variable names - no variable shadowing errors")
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
