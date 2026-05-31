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
	events   []eventHandler
	children []any
	void     bool
}

// NewElement creates an Element with the given HTML tag.
// Used by tinywasm/html, tinywasm/svg, tinywasm/image to build elements.
func NewElement(tag string) *Element { return &Element{tag: tag} }

// NoCloseTag marks the element as self-closing (no closing tag rendered).
// Use for void HTML elements: br, hr, img, input, link, meta, etc.
func (b *Element) NoCloseTag() *Element {
	b.void = true
	return b
}

// ID sets the ID of the element.
func (b *Element) ID(id string) *Element {
	b.id = id
	return b
}

// For sets the for= attribute pointing to other's ID, auto-generating
// other's ID if it has none. Use for label/input pairing and aria-* references.
func (b *Element) For(other *Element) *Element {
	if other == nil {
		return b
	}
	return b.Attr("for", other.GetID())
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

// On adds a generic event handler.
func (b *Element) On(t string, h func(Event)) *Element {
	b.events = append(b.events, eventHandler{Name: t, Handler: h})
	return b
}

// Add adds one or more children or attributes to the element.
// Children can be *Element, Component, string, or fmt.KeyValue.
func (b *Element) Add(children ...any) *Element {
	for _, child := range children {
		if attr, ok := child.(fmt.KeyValue); ok {
			switch attr.Key {
			case "class":
				b.Class(attr.Value)
			case "id":
				b.ID(attr.Value)
			default:
				b.Attr(attr.Key, attr.Value)
			}
			continue
		}
		b.children = append(b.children, child)
	}
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

// Update triggers a re-render of the component.
func (b *Element) Update() {
	Update(b)
}

// --- Component Interface Implementation ---

// GetID returns the element's ID.
func (b *Element) GetID() string {
	if b.id == "" {
		b.id = generateID()
	}
	return b.id
}

// SetID sets the element's ID.
func (b *Element) SetID(id string) {
	b.id = id
}

// String serializes the element tree to its string representation.
func (b *Element) String() string {
	return elementToHTML(b)
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

// Helper to convert Element to HTML string (recursive)
func elementToHTML(el *Element) string {
	s := "<" + el.tag
	if el.id != "" {
		s += " id='" + el.id + "'"
	}
	if len(el.classes) > 0 {
		s += " class='"
		for i, c := range el.classes {
			if i > 0 {
				s += " "
			}
			s += c
		}
		s += "'"
	}
	for _, attr := range el.attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
	s += ">"
	if el.void {
		return s // No children, no closing tag
	}
	for _, child := range el.children {
		switch v := child.(type) {
		case *Element:
			s += elementToHTML(v)
		case string:
			s += v
		case Component:
			s += v.String()
		default:
			s += fmt.Sprint(v)
		}
	}
	s += "</" + el.tag + ">"
	return s
}

