package trackby

import "github.com/vcrobe/nojs/runtime"

// Item represents a data item with ID for trackBy
type Item struct {
	ID   int
	Name string
}

// MultiItemList demonstrates trackBy with multiple sibling child elements per loop iteration.
// This tests a critical compiler edge case where multiple children in a loop body
// previously caused variable shadowing errors.
type MultiItemList struct {
	runtime.ComponentBase
	Items []Item
}

func (m *MultiItemList) OnInit() {
	m.Items = []Item{
		{ID: 101, Name: "Alpha"},
		{ID: 102, Name: "Beta"},
		{ID: 103, Name: "Gamma"},
	}
}

func (m *MultiItemList) AddItem(name string) {
	newID := 100 + len(m.Items) + 1
	m.Items = append(m.Items, Item{
		ID:   newID,
		Name: name,
	})
	m.StateHasChanged()
}

func (m *MultiItemList) ClearItems() {
	m.Items = []Item{}
	m.StateHasChanged()
}
