package trackby

import "github.com/ForgeLogic/nojs/runtime"

// TagList demonstrates trackBy with primitive string slice
type TagList struct {
	runtime.ComponentBase
	Tags []string
}

func (t *TagList) OnMount() {
	t.Tags = []string{"golang", "wasm", "component", "framework"}
}

func (t *TagList) AddTag(newTag string) {
	t.Tags = append(t.Tags, newTag)
	t.StateHasChanged()
}

func (t *TagList) ClearTags() {
	t.Tags = []string{}
	t.StateHasChanged()
}
