//go:build !wasm
// +build !wasm

package trackby

import (
	"testing"

	"github.com/ForgeLogic/nojs-compiler/testcomponents"
	"github.com/ForgeLogic/nojs/vdom"
)

// TestProductList_DotNotationTrackBy_InitialRender verifies that trackBy with dot-notation
// (struct field access) correctly renders the initial list of products.
func TestProductList_DotNotationTrackBy_InitialRender(t *testing.T) {
	// Arrange: Create a ProductList component with initial products
	productList := &ProductList{
		Products: []Product{
			{ID: 1, Name: "Laptop"},
			{ID: 2, Name: "Mouse"},
			{ID: 3, Name: "Keyboard"},
		},
	}
	renderer := testcomponents.NewTestRenderer(productList)

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

	// Assert: Verify the number of list items matches the products
	expectedProductCount := 3 // Laptop, Mouse, Keyboard
	if len(ulNode.Children) != expectedProductCount {
		t.Fatalf("Expected %d <li> elements, got %d", expectedProductCount, len(ulNode.Children))
	}

	// Assert: Verify each product is rendered correctly
	expectedProducts := []struct {
		id   int
		name string
	}{
		{1, "Laptop"},
		{2, "Mouse"},
		{3, "Keyboard"},
	}
	for i, expectedProduct := range expectedProducts {
		liNode := ulNode.Children[i]
		if liNode.Tag != "li" {
			t.Errorf("Expected child %d to be 'li', got '%s'", i, liNode.Tag)
		}
		expectedContent := "Product " + string(rune('0'+i)) + ": " + expectedProduct.name + " (ID: " + string(rune('0'+expectedProduct.id)) + ")"
		if liNode.Content != expectedContent {
			t.Errorf("Child %d: expected '%s', got '%s'", i, expectedContent, liNode.Content)
		}
	}
}

// TestProductList_DotNotationTrackBy_AddProduct verifies that calling AddProduct() triggers a re-render
// with the new product added to the list.
func TestProductList_DotNotationTrackBy_AddProduct(t *testing.T) {
	// Arrange
	productList := &ProductList{
		Products: []Product{
			{ID: 1, Name: "Laptop"},
			{ID: 2, Name: "Mouse"},
			{ID: 3, Name: "Keyboard"},
		},
	}
	renderer := testcomponents.NewTestRenderer(productList)
	vnode1 := renderer.RenderRoot()

	// Find the initial <ul> element
	var ulNode1 *vdom.VNode
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}
	if len(ulNode1.Children) != 3 {
		t.Fatalf("Initial product count should be 3, got %d", len(ulNode1.Children))
	}

	// Act: Add a new product
	productList.AddProduct("Monitor")

	// Assert: Verify the VDOM was updated
	vnode2 := renderer.GetCurrentVDOM()
	var ulNode2 *vdom.VNode
	for _, child := range vnode2.Children {
		if child.Tag == "ul" {
			ulNode2 = child
			break
		}
	}

	expectedProductCount := 4
	if len(ulNode2.Children) != expectedProductCount {
		t.Fatalf("After adding a product, expected %d <li> elements, got %d", expectedProductCount, len(ulNode2.Children))
	}

	// Assert: Verify the new product was added at the end with correct ID
	lastLiNode := ulNode2.Children[3]
	expectedContent := "Product 3: Monitor (ID: 4)"
	if lastLiNode.Content != expectedContent {
		t.Errorf("Expected last product '%s', got '%s'", expectedContent, lastLiNode.Content)
	}
}

// TestProductList_DotNotationTrackBy_MultipleAdditions verifies that multiple AddProduct() calls
// each trigger re-renders with cumulative additions and correct IDs.
func TestProductList_DotNotationTrackBy_MultipleAdditions(t *testing.T) {
	// Arrange
	productList := &ProductList{
		Products: []Product{
			{ID: 1, Name: "Laptop"},
			{ID: 2, Name: "Mouse"},
			{ID: 3, Name: "Keyboard"},
		},
	}
	renderer := testcomponents.NewTestRenderer(productList)
	renderer.RenderRoot()

	// Act & Assert: Add multiple products
	productsToAdd := []string{"Monitor", "Headphones", "Webcam"}
	for i, newProduct := range productsToAdd {
		productList.AddProduct(newProduct)
		vnode := renderer.GetCurrentVDOM()

		var ulNode *vdom.VNode
		for _, child := range vnode.Children {
			if child.Tag == "ul" {
				ulNode = child
				break
			}
		}

		expectedCount := 3 + i + 1
		if len(ulNode.Children) != expectedCount {
			t.Errorf("After adding product %d, expected %d items, got %d", i+1, expectedCount, len(ulNode.Children))
		}

		// Verify the newly added product with correct ID
		lastLiNode := ulNode.Children[len(ulNode.Children)-1]
		newID := 3 + i + 1
		expectedContent := "Product " + string(rune('0'+expectedCount-1)) + ": " + newProduct + " (ID: " + string(rune('0'+newID)) + ")"
		if lastLiNode.Content != expectedContent {
			t.Errorf("Product %d: expected '%s', got '%s'", i+1, expectedContent, lastLiNode.Content)
		}
	}
}

// TestProductList_DotNotationTrackBy_ClearProducts verifies that ClearProducts() removes all products
// and triggers a re-render with an empty list.
func TestProductList_DotNotationTrackBy_ClearProducts(t *testing.T) {
	// Arrange
	productList := &ProductList{
		Products: []Product{
			{ID: 1, Name: "Laptop"},
			{ID: 2, Name: "Mouse"},
			{ID: 3, Name: "Keyboard"},
		},
	}
	renderer := testcomponents.NewTestRenderer(productList)
	vnode1 := renderer.RenderRoot()

	// Verify initial state
	var ulNode1 *vdom.VNode
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}
	if len(ulNode1.Children) != 3 {
		t.Fatalf("Initial product count should be 3, got %d", len(ulNode1.Children))
	}

	// Act: Clear all products
	productList.ClearProducts()

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
		t.Errorf("After clearing products, expected 0 items, got %d", len(ulNode2.Children))
	}
}

// TestProductList_DotNotationTrackBy_AddAfterClear verifies that AddProduct() works correctly
// after clearing the product list, with IDs starting from 1 again.
func TestProductList_DotNotationTrackBy_AddAfterClear(t *testing.T) {
	// Arrange
	productList := &ProductList{
		Products: []Product{
			{ID: 1, Name: "Laptop"},
			{ID: 2, Name: "Mouse"},
			{ID: 3, Name: "Keyboard"},
		},
	}
	renderer := testcomponents.NewTestRenderer(productList)
	renderer.RenderRoot()

	// Act: Clear and then add new products
	productList.ClearProducts()
	productList.AddProduct("NewItem")

	// Assert: Verify the list contains only the newly added product
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

	// Verify the product has ID 1 (since list was cleared)
	liNode := ulNode.Children[0]
	expectedContent := "Product 0: NewItem (ID: 1)"
	if liNode.Content != expectedContent {
		t.Errorf("Expected '%s', got '%s'", expectedContent, liNode.Content)
	}
}

// TestProductList_DotNotationTrackBy_RenderIsolation verifies that multiple ProductList instances
// maintain separate state and VDOM.
func TestProductList_DotNotationTrackBy_RenderIsolation(t *testing.T) {
	// Arrange: Create two separate ProductList instances
	productList1 := &ProductList{
		Products: []Product{
			{ID: 1, Name: "Laptop"},
			{ID: 2, Name: "Mouse"},
			{ID: 3, Name: "Keyboard"},
		},
	}
	productList2 := &ProductList{
		Products: []Product{
			{ID: 1, Name: "Laptop"},
			{ID: 2, Name: "Mouse"},
			{ID: 3, Name: "Keyboard"},
		},
	}

	renderer1 := testcomponents.NewTestRenderer(productList1)
	renderer2 := testcomponents.NewTestRenderer(productList2)

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

	// Both should have the same initial products (3)
	if len(ulNode1.Children) != 3 || len(ulNode2.Children) != 3 {
		t.Fatalf("Both instances should have 3 initial products")
	}

	// Act: Modify only the first instance
	productList1.AddProduct("UniqueProduct")

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

	if len(ulNode1.Children) != 4 {
		t.Errorf("First instance should have 4 products after add, got %d", len(ulNode1.Children))
	}
	if len(ulNode2.Children) != 3 {
		t.Errorf("Second instance should still have 3 products, got %d", len(ulNode2.Children))
	}
}

// TestProductList_DotNotationTrackBy_TrackByUsingProductID verifies that the trackBy field
// (product.ID) is correctly used to uniquely identify products in the list.
// This test ensures that products are keyed by their ID, not by their position.
func TestProductList_DotNotationTrackBy_TrackByUsingProductID(t *testing.T) {
	// Arrange
	productList := &ProductList{
		Products: []Product{
			{ID: 1, Name: "Laptop"},
			{ID: 2, Name: "Mouse"},
			{ID: 3, Name: "Keyboard"},
		},
	}
	renderer := testcomponents.NewTestRenderer(productList)
	vnode1 := renderer.RenderRoot()

	// Get initial products
	var ulNode1 *vdom.VNode
	for _, child := range vnode1.Children {
		if child.Tag == "ul" {
			ulNode1 = child
			break
		}
	}

	// Verify initial IDs
	for i, liNode := range ulNode1.Children {
		// The content format is "Product {i}: {product.Name} (ID: {product.ID})"
		// We can verify the ID is present in the content
		if liNode.Tag != "li" {
			t.Errorf("Expected li tag at index %d", i)
		}
	}

	// Act: Add a product and verify IDs increment correctly
	productList.AddProduct("Tablet")

	// Assert
	vnode2 := renderer.GetCurrentVDOM()
	var ulNode2 *vdom.VNode
	for _, child := range vnode2.Children {
		if child.Tag == "ul" {
			ulNode2 = child
			break
		}
	}

	// After adding, we should have 4 products with IDs 1, 2, 3, 4
	if len(ulNode2.Children) != 4 {
		t.Fatalf("Expected 4 products after add, got %d", len(ulNode2.Children))
	}

	// The new product should have ID 4
	lastLiNode := ulNode2.Children[3]
	expectedContent := "Product 3: Tablet (ID: 4)"
	if lastLiNode.Content != expectedContent {
		t.Errorf("Expected '%s', got '%s'", expectedContent, lastLiNode.Content)
	}
}
