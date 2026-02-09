package dom

import (
	"github.com/tinywasm/fmt"
)

// Builder represents a DOM element in the fluent builder API.
// It matches the "Element" struct concept in the prompt but renamed to avoid conflict with Element interface.
type Builder struct {
	tag      string
	id       string
	classes  []string
	attrs    map[string]string
	events   []EventHandler
	children []any
}

// ID sets the ID of the element.
func (b *Builder) ID(id string) *Builder {
	b.id = id
	return b
}

// Class adds a class to the element.
func (b *Builder) Class(class string) *Builder {
	b.classes = append(b.classes, class)
	return b
}

// Attr sets an attribute on the element.
func (b *Builder) Attr(key, val string) *Builder {
	if b.attrs == nil {
		b.attrs = make(map[string]string)
	}
	b.attrs[key] = val
	return b
}

// OnClick adds a click event handler.
func (b *Builder) OnClick(handler func(Event)) *Builder {
	b.events = append(b.events, EventHandler{"click", handler})
	return b
}

// OnInput adds an input event handler.
func (b *Builder) OnInput(handler func(Event)) *Builder {
	b.events = append(b.events, EventHandler{"input", handler})
	return b
}

// OnChange adds a change event handler.
func (b *Builder) OnChange(handler func(Event)) *Builder {
	b.events = append(b.events, EventHandler{"change", handler})
	return b
}

// Append adds a child to the element.
func (b *Builder) Append(child any) *Builder {
	b.children = append(b.children, child)
	return b
}

// Text adds a text node child.
func (b *Builder) Text(text string) *Builder {
	b.children = append(b.children, text)
	return b
}

// Render renders the element to the parent.
// This is a terminal operation.
func (b *Builder) Render(parentID string) error {
	return Render(parentID, b)
}

// Mount is an alias for Render.
func (b *Builder) Mount(parentID string) error {
	return Render(parentID, b)
}

// ToNode converts the element to a Node tree.
func (b *Builder) ToNode() Node {
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
	for k, v := range b.attrs {
		attrs = append(attrs, fmt.KeyValue{Key: k, Value: v})
	}

	// Convert children
	var children []any
	for _, child := range b.children {
		switch c := child.(type) {
		case *Builder:
			children = append(children, c.ToNode())
		case Builder:
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
func (b *Builder) GetID() string {
	return b.id
}

// SetID sets the element's ID.
func (b *Builder) SetID(id string) {
	b.id = id
}

// RenderHTML renders the element to HTML string.
func (b *Builder) RenderHTML() string {
	return nodeToHTML(b.ToNode())
}

// Children returns the component's children (components only).
func (b *Builder) Children() []Component {
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
func Div() *Builder    { return &Builder{tag: "div"} }
func Span() *Builder   { return &Builder{tag: "span"} }
func Button() *Builder { return &Builder{tag: "button"} }
func H1() *Builder     { return &Builder{tag: "h1"} }
func H2() *Builder     { return &Builder{tag: "h2"} }
func H3() *Builder     { return &Builder{tag: "h3"} }
func P() *Builder      { return &Builder{tag: "p"} }
func Ul() *Builder     { return &Builder{tag: "ul"} }
func Li() *Builder     { return &Builder{tag: "li"} }
func Input() *Builder  { return &Builder{tag: "input"} }
func Form() *Builder   { return &Builder{tag: "form"} }
func A() *Builder      { return &Builder{tag: "a"} }
func Img() *Builder    { return &Builder{tag: "img"} }

// Helper for backward compatibility (moved from html/builder.go)
// Tag creates a new Node.
func Tag(tag string, children ...any) Node {
	n := Node{
		Tag: tag,
	}
	for _, child := range children {
		switch v := child.(type) {
		case Node:
			n.Children = append(n.Children, v)
		case string:
			n.Children = append(n.Children, v)
		case Component:
			n.Children = append(n.Children, v)
		case func(Event):
			// Ignore raw functions in old API or handle if possible
		case attr:
			n.Attrs = append(n.Attrs, fmt.KeyValue{Key: v.key, Value: v.val})
		case event:
			n.Events = append(n.Events, EventHandler{Name: v.name, Handler: v.handler})
		}
	}
	return n
}

// Helper types for Tag function
type attr struct {
	key string
	val string
}

type event struct {
	name    string
	handler func(Event)
}

// Helpers for Tag function
func TAttr(k, v string) attr   { return attr{k, v} }
func ID(id string) attr       { return attr{"id", id} }
func Class(class string) attr { return attr{"class", class} }
func OnClick(h func(Event)) event  { return event{"click", h} }
func OnInput(h func(Event)) event  { return event{"input", h} }
func OnChange(h func(Event)) event { return event{"change", h} }
func Text(s string) string { return s }
