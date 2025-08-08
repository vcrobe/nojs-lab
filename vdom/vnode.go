package vdom

// VNode represents a virtual DOM node.
type VNode struct {
	Tag        string         // The HTML tag name
	Attributes map[string]any // The attributes of the node
	Children   []*VNode       // The child nodes
	Content    string         // The content of the node
	OnClick    func()         // Optional click event handler
}

// NewVNode creates a new VNode.
func NewVNode(tag string, attributes map[string]any, children []*VNode, content string) *VNode {
	var onClick func()
	if attributes != nil {
		if v, ok := attributes["onClick"]; ok {
			if f, ok := v.(func()); ok {
				onClick = f
				// Remove from attributes so it doesn't get rendered as an HTML attribute
				delete(attributes, "onClick")
			}
		}
	}
	return &VNode{
		Tag:        tag,
		Attributes: attributes,
		Children:   children,
		Content:    content,
		OnClick:    onClick,
	}
}

// SetContent updates the Content field of the VNode.
func (v *VNode) SetContent(content string) {
	v.Content = content
}

// Paragraph creates a <p> VNode with the given text as its child and allows passing attributes.
func Paragraph(text string, attrs map[string]any) *VNode {
	return NewVNode("p", attrs, nil, text)
}

// InputText returns a VNode representing an <input type="text"> element.
// Optionally accepts a map of attributes (e.g., {"placeholder": "Type here"}).
func InputText(attrs map[string]any) *VNode {
	if attrs == nil {
		attrs = make(map[string]any)
	}
	attrs["type"] = "text"
	return NewVNode("input", attrs, nil, "")
}

// Div creates a <div> VNode with the given children and allows passing attributes.
func Div(attrs map[string]any, children ...*VNode) *VNode {
	return NewVNode("div", attrs, children, "")
}

// Button creates a <button> VNode with the given children and allows passing attributes.
func Button(content string, attrs map[string]any, children ...*VNode) *VNode {
	return NewVNode("button", attrs, children, content)
}
