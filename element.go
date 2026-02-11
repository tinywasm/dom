package dom

import (
	"github.com/tinywasm/fmt"
)

// Element represents a DOM element in the fluent Element API.
type Element struct {
	tag      string
	id       string
	classes  []string
	attrs    []fmt.KeyValue
	events   []EventHandler
	children []any
}

// ID sets the ID of the element.
func (b *Element) ID(id string) *Element {
	b.id = id
	return b
}

// Class adds a class to the element.
func (b *Element) Class(class ...string) *Element {
	b.classes = append(b.classes, class...)
	return b
}

// Attr sets an attribute on the element.
func (b *Element) Attr(key, val string) *Element {
	for i, attr := range b.attrs {
		if attr.Key == key {
			b.attrs[i].Value = val
			return b
		}
	}
	b.attrs = append(b.attrs, fmt.KeyValue{Key: key, Value: val})
	return b
}

// OnClick adds a click event handler.
func (b *Element) OnClick(handler func(Event)) *Element {
	b.events = append(b.events, EventHandler{"click", handler})
	return b
}

// OnInput adds an input event handler.
func (b *Element) OnInput(handler func(Event)) *Element {
	b.events = append(b.events, EventHandler{"input", handler})
	return b
}

// OnChange adds a change event handler.
func (b *Element) OnChange(handler func(Event)) *Element {
	b.events = append(b.events, EventHandler{"change", handler})
	return b
}

// Add adds one or more children to the element.
// Children can be *Element, Node, Component, or string.
func (b *Element) Add(children ...any) *Element {
	b.children = append(b.children, children...)
	return b
}

// Append adds a child to the element.
// Deprecated: use Add instead.
func (b *Element) Append(child any) *Element {
	b.children = append(b.children, child)
	return b
}

// Text adds a text node child.
func (b *Element) Text(text string) *Element {
	b.children = append(b.children, text)
	return b
}

// Render renders the element to the parent.
// This is a terminal operation.
func (b *Element) Render(parentID string) error {
	return Render(parentID, b)
}

// Mount is an alias for Render.
func (b *Element) Mount(parentID string) error {
	return Render(parentID, b)
}

// ToNode converts the element to a Node tree.
func (b *Element) ToNode() Node {
	var attrs []fmt.KeyValue
	if b.id != "" {
		attrs = append(attrs, fmt.KeyValue{Key: "id", Value: b.id})
	}
	if len(b.classes) > 0 {
		classStr := ""
		for i, c := range b.classes {
			if i > 0 {
				classStr += " "
			}
			classStr += c
		}
		attrs = append(attrs, fmt.KeyValue{Key: "class", Value: classStr})
	}
	attrs = append(attrs, b.attrs...)

	// Convert children
	var children []any
	for _, child := range b.children {
		switch c := child.(type) {
		case *Element:
			children = append(children, c.ToNode())
		case Element:
			children = append(children, c.ToNode())
		default:
			children = append(children, c)
		}
	}

	return Node{
		Tag:      b.tag,
		Attrs:    attrs,
		Events:   b.events,
		Children: children,
	}
}

// --- Component Interface Implementation ---

// GetID returns the element's ID.
func (b *Element) GetID() string {
	return b.id
}

// SetID sets the element's ID.
func (b *Element) SetID(id string) {
	b.id = id
}

// RenderHTML renders the element to HTML string.
func (b *Element) RenderHTML() string {
	return nodeToHTML(b.ToNode())
}

// Children returns the component's children (components only).
func (b *Element) Children() []Component {
	var comps []Component
	for _, child := range b.children {
		if c, ok := child.(Component); ok {
			comps = append(comps, c)
		}
	}
	return comps
}

// Helper to convert Node to HTML string (recursive)
func nodeToHTML(n Node) string {
	s := "<" + n.Tag
	for _, attr := range n.Attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
	s += ">"
	for _, child := range n.Children {
		switch v := child.(type) {
		case Node:
			s += nodeToHTML(v)
		case string:
			s += v
		case Component:
			s += v.RenderHTML()
		default:
			// Fallback for other types if any
			s += fmt.Sprint(v)
		}
	}
	s += "</" + n.Tag + ">"
	return s
}

// Factory functions
func Div() *Element    { return &Element{tag: "div"} }
func Span() *Element   { return &Element{tag: "span"} }
func Button() *Element { return &Element{tag: "button"} }
func H1() *Element     { return &Element{tag: "h1"} }
func H2() *Element     { return &Element{tag: "h2"} }
func H3() *Element     { return &Element{tag: "h3"} }
func P() *Element      { return &Element{tag: "p"} }
func Ul() *Element     { return &Element{tag: "ul"} }
func Li() *Element     { return &Element{tag: "li"} }
func Input() *Element  { return &Element{tag: "input"} }
func Form() *Element   { return &Element{tag: "form"} }
func A() *Element      { return &Element{tag: "a"} }
func Img() *Element    { return &Element{tag: "img"} }
