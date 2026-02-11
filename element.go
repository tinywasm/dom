package dom

import (
	"github.com/tinywasm/fmt"
)

// Element represents a DOM element in the declarative API.
type Element struct {
	tag      string
	id       string
	classes  []string
	attrs    []fmt.KeyValue
	events   []EventHandler
	children []any // Accepts: *Element, string, Component
}

// ID sets the ID of the element.
func (e *Element) ID(id string) *Element {
	e.id = id
	return e
}

// Class adds one or more classes to the element.
func (e *Element) Class(classes ...string) *Element {
	e.classes = append(e.classes, classes...)
	return e
}

// Attr sets an attribute on the element.
func (e *Element) Attr(key, val string) *Element {
	for i, attr := range e.attrs {
		if attr.Key == key {
			e.attrs[i].Value = val
			return e
		}
	}
	e.attrs = append(e.attrs, fmt.KeyValue{Key: key, Value: val})
	return e
}

// OnClick adds a click event handler.
func (e *Element) OnClick(handler func(Event)) *Element {
	e.events = append(e.events, EventHandler{"click", handler})
	return e
}

// OnInput adds an input event handler.
func (e *Element) OnInput(handler func(Event)) *Element {
	e.events = append(e.events, EventHandler{"input", handler})
	return e
}

// OnChange adds a change event handler.
func (e *Element) OnChange(handler func(Event)) *Element {
	e.events = append(e.events, EventHandler{"change", handler})
	return e
}

// On adds a generic event handler.
func (e *Element) On(eventType string, handler func(Event)) *Element {
	e.events = append(e.events, EventHandler{eventType, handler})
	return e
}

// Add adds one or more children to the element.
// Children can be *Element, Node, Component, or string.
func (e *Element) Add(children ...any) *Element {
	e.children = append(e.children, children...)
	return e
}

// Text adds a text node child.
func (e *Element) Text(text string) *Element {
	e.children = append(e.children, text)
	return e
}

// Render renders the element to the parent.
// This is a terminal operation.
func (e *Element) Render(parentID string) error {
	return Render(parentID, e)
}

// ToNode converts the element to a Node tree.
func (e *Element) ToNode() Node {
	var attrs []fmt.KeyValue
	if e.id != "" {
		attrs = append(attrs, fmt.KeyValue{Key: "id", Value: e.id})
	}
	if len(e.classes) > 0 {
		classStr := ""
		for i, c := range e.classes {
			if i > 0 {
				classStr += " "
			}
			classStr += c
		}
		attrs = append(attrs, fmt.KeyValue{Key: "class", Value: classStr})
	}
	attrs = append(attrs, e.attrs...)

	// Convert children
	var children []any
	for _, child := range e.children {
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
		Tag:      e.tag,
		Attrs:    attrs,
		Events:   e.events,
		Children: children,
	}
}

// --- Component Interface Implementation ---

// GetID returns the element's ID.
func (e *Element) GetID() string {
	return e.id
}

// SetID sets the element's ID.
func (e *Element) SetID(id string) {
	e.id = id
}

// RenderHTML renders the element to HTML string.
func (e *Element) RenderHTML() string {
	return nodeToHTML(e.ToNode())
}

// Children returns the component's children (components only).
func (e *Element) Children() []Component {
	var comps []Component
	for _, child := range e.children {
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
