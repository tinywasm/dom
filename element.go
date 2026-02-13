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
	void     bool // NEW: self-closing element, no children/closing tag
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

// On adds a generic event handler.
func (b *Element) On(t string, h func(Event)) *Element {
	b.events = append(b.events, eventHandler{Name: t, Handler: h})
	return b
}

// Add adds one or more children to the element.
// Children can be *Element, Node, Component, or string.
func (b *Element) Add(children ...any) *Element {
	b.children = append(b.children, children...)
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
func (b *Element) Update() error {
	return Update(b)
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

// RenderHTML renders the element to HTML string.
func (b *Element) RenderHTML() string {
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
			s += v.RenderHTML()
		default:
			s += fmt.Sprint(v)
		}
	}
	s += "</" + el.tag + ">"
	return s
}

// Factory functions
func Div(children ...any) *Element        { return &Element{tag: "div", children: children} }
func Span(children ...any) *Element       { return &Element{tag: "span", children: children} }
func P(children ...any) *Element          { return &Element{tag: "p", children: children} }
func H1(children ...any) *Element         { return &Element{tag: "h1", children: children} }
func H2(children ...any) *Element         { return &Element{tag: "h2", children: children} }
func H3(children ...any) *Element         { return &Element{tag: "h3", children: children} }
func H4(children ...any) *Element         { return &Element{tag: "h4", children: children} }
func H5(children ...any) *Element         { return &Element{tag: "h5", children: children} }
func H6(children ...any) *Element         { return &Element{tag: "h6", children: children} }
func Ul(children ...any) *Element         { return &Element{tag: "ul", children: children} }
func Ol(children ...any) *Element         { return &Element{tag: "ol", children: children} }
func Li(children ...any) *Element         { return &Element{tag: "li", children: children} }
func Nav(children ...any) *Element        { return &Element{tag: "nav", children: children} }
func Section(children ...any) *Element    { return &Element{tag: "section", children: children} }
func Main(children ...any) *Element       { return &Element{tag: "main", children: children} }
func Article(children ...any) *Element    { return &Element{tag: "article", children: children} }
func Header(children ...any) *Element     { return &Element{tag: "header", children: children} }
func Footer(children ...any) *Element     { return &Element{tag: "footer", children: children} }
func Aside(children ...any) *Element      { return &Element{tag: "aside", children: children} }
func Details(children ...any) *Element    { return &Element{tag: "details", children: children} }
func Summary(children ...any) *Element    { return &Element{tag: "summary", children: children} }
func Dialog(children ...any) *Element     { return &Element{tag: "dialog", children: children} }
func Figure(children ...any) *Element     { return &Element{tag: "figure", children: children} }
func Figcaption(children ...any) *Element { return &Element{tag: "figcaption", children: children} }
func Pre(children ...any) *Element        { return &Element{tag: "pre", children: children} }
func Code(children ...any) *Element       { return &Element{tag: "code", children: children} }
func Strong(children ...any) *Element     { return &Element{tag: "strong", children: children} }
func Em(children ...any) *Element         { return &Element{tag: "em", children: children} }
func Small(children ...any) *Element      { return &Element{tag: "small", children: children} }
func Mark(children ...any) *Element       { return &Element{tag: "mark", children: children} }
func Table(children ...any) *Element      { return &Element{tag: "table", children: children} }
func Thead(children ...any) *Element      { return &Element{tag: "thead", children: children} }
func Tbody(children ...any) *Element      { return &Element{tag: "tbody", children: children} }
func Tfoot(children ...any) *Element      { return &Element{tag: "tfoot", children: children} }
func Tr(children ...any) *Element         { return &Element{tag: "tr", children: children} }
func Th(children ...any) *Element         { return &Element{tag: "th", children: children} }
func Td(children ...any) *Element         { return &Element{tag: "td", children: children} }
func Fieldset(children ...any) *Element   { return &Element{tag: "fieldset", children: children} }
func Legend(children ...any) *Element     { return &Element{tag: "legend", children: children} }
func Label(children ...any) *Element      { return &Element{tag: "label", children: children} }
func Canvas(children ...any) *Element     { return &Element{tag: "canvas", children: children} }
func Style(children ...any) *Element      { return &Element{tag: "style", children: children} }
func Script(children ...any) *Element     { return &Element{tag: "script", children: children} }

// Enhanced factories with key attrs as args
func A(href string, children ...any) *Element {
	return &Element{tag: "a", children: children,
		attrs: []fmt.KeyValue{{Key: "href", Value: href}}}
}
func Button(children ...any) *Element { return &Element{tag: "button", children: children} }

// Void element factories (void: true, no children)
func Img(src, alt string) *Element {
	return &Element{tag: "img", void: true,
		attrs: []fmt.KeyValue{{Key: "src", Value: src}, {Key: "alt", Value: alt}}}
}
func Br() *Element { return &Element{tag: "br", void: true} }
func Hr() *Element { return &Element{tag: "hr", void: true} }

// Option helpers
func Option(value, text string) *Element {
	return &Element{tag: "option", children: []any{text},
		attrs: []fmt.KeyValue{{Key: "value", Value: value}}}
}
func SelectedOption(value, text string) *Element {
	return &Element{tag: "option", children: []any{text},
		attrs: []fmt.KeyValue{{Key: "value", Value: value}, {Key: "selected", Value: ""}}}
}
