package trackby

import "github.com/ForgeLogic/nojs/runtime"

// Product represents an item with an ID for trackBy
type Product struct {
	ID   int
	Name string
}

// ProductList demonstrates trackBy with struct slice using dot-notation
type ProductList struct {
	runtime.ComponentBase
	Products []Product
}

func (p *ProductList) OnMount() {
	p.Products = []Product{
		{ID: 1, Name: "Laptop"},
		{ID: 2, Name: "Mouse"},
		{ID: 3, Name: "Keyboard"},
	}
}

func (p *ProductList) AddProduct(name string) {
	newID := len(p.Products) + 1
	p.Products = append(p.Products, Product{
		ID:   newID,
		Name: name,
	})
	p.StateHasChanged()
}

func (p *ProductList) ClearProducts() {
	p.Products = []Product{}
	p.StateHasChanged()
}
