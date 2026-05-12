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

func Svg(children ...any) *Element { return (&Element{tag: "svg"}).Add(children...) }
func Use(children ...any) *Element { return (&Element{tag: "use"}).Add(children...) }

func Div(children ...any) *Element        { return (&Element{tag: "div"}).Add(children...) }
func Span(children ...any) *Element       { return (&Element{tag: "span"}).Add(children...) }
func P(children ...any) *Element          { return (&Element{tag: "p"}).Add(children...) }
func H1(children ...any) *Element         { return (&Element{tag: "h1"}).Add(children...) }
func H2(children ...any) *Element         { return (&Element{tag: "h2"}).Add(children...) }
func H3(children ...any) *Element         { return (&Element{tag: "h3"}).Add(children...) }
func H4(children ...any) *Element         { return (&Element{tag: "h4"}).Add(children...) }
func H5(children ...any) *Element         { return (&Element{tag: "h5"}).Add(children...) }
func H6(children ...any) *Element         { return (&Element{tag: "h6"}).Add(children...) }
func Ul(children ...any) *Element         { return (&Element{tag: "ul"}).Add(children...) }
func Ol(children ...any) *Element         { return (&Element{tag: "ol"}).Add(children...) }
func Li(children ...any) *Element         { return (&Element{tag: "li"}).Add(children...) }
func Nav(children ...any) *Element        { return (&Element{tag: "nav"}).Add(children...) }
func Section(children ...any) *Element    { return (&Element{tag: "section"}).Add(children...) }
func Main(children ...any) *Element       { return (&Element{tag: "main"}).Add(children...) }
func Article(children ...any) *Element    { return (&Element{tag: "article"}).Add(children...) }
func Header(children ...any) *Element     { return (&Element{tag: "header"}).Add(children...) }
func Footer(children ...any) *Element     { return (&Element{tag: "footer"}).Add(children...) }
func Aside(children ...any) *Element      { return (&Element{tag: "aside"}).Add(children...) }
func Details(children ...any) *Element    { return (&Element{tag: "details"}).Add(children...) }
func Summary(children ...any) *Element    { return (&Element{tag: "summary"}).Add(children...) }
func Dialog(children ...any) *Element     { return (&Element{tag: "dialog"}).Add(children...) }
func Figure(children ...any) *Element     { return (&Element{tag: "figure"}).Add(children...) }
func Figcaption(children ...any) *Element { return (&Element{tag: "figcaption"}).Add(children...) }
func Pre(children ...any) *Element        { return (&Element{tag: "pre"}).Add(children...) }
func Code(children ...any) *Element       { return (&Element{tag: "code"}).Add(children...) }
func Strong(children ...any) *Element     { return (&Element{tag: "strong"}).Add(children...) }
func Em(children ...any) *Element         { return (&Element{tag: "em"}).Add(children...) }
func Small(children ...any) *Element      { return (&Element{tag: "small"}).Add(children...) }
func Mark(children ...any) *Element       { return (&Element{tag: "mark"}).Add(children...) }
func Table(children ...any) *Element      { return (&Element{tag: "table"}).Add(children...) }
func Thead(children ...any) *Element      { return (&Element{tag: "thead"}).Add(children...) }
func Tbody(children ...any) *Element      { return (&Element{tag: "tbody"}).Add(children...) }
func Tfoot(children ...any) *Element      { return (&Element{tag: "tfoot"}).Add(children...) }
func Tr(children ...any) *Element         { return (&Element{tag: "tr"}).Add(children...) }
func Th(children ...any) *Element         { return (&Element{tag: "th"}).Add(children...) }
func Td(children ...any) *Element         { return (&Element{tag: "td"}).Add(children...) }
func Fieldset(children ...any) *Element   { return (&Element{tag: "fieldset"}).Add(children...) }
func Legend(children ...any) *Element     { return (&Element{tag: "legend"}).Add(children...) }
func Label(children ...any) *Element      { return (&Element{tag: "label"}).Add(children...) }
func Canvas(children ...any) *Element     { return (&Element{tag: "canvas"}).Add(children...) }
func Style(children ...any) *Element      { return (&Element{tag: "style"}).Add(children...) }
func Script(children ...any) *Element     { return (&Element{tag: "script"}).Add(children...) }

// Enhanced factories with key attrs as args
func A(href string, children ...any) *Element {
	return (&Element{tag: "a"}).Attr("href", href).Add(children...)
}
func Button(children ...any) *Element { return (&Element{tag: "button"}).Add(children...) }

// Void element factories (void: true, no children)
func Img(src, alt string) *Element {
	return (&Element{tag: "img", void: true}).Attr("src", src).Attr("alt", alt)
}
func Br() *Element { return &Element{tag: "br", void: true} }
func Hr() *Element { return &Element{tag: "hr", void: true} }
func Input(typ string) *Element {
	return (&Element{tag: "input", void: true}).Attr("type", typ)
}

// Option helpers
func Option(value, text string) *Element {
	return (&Element{tag: "option"}).Attr("value", value).Add(text)
}
func SelectedOption(value, text string) *Element {
	return (&Element{tag: "option"}).Attr("value", value).Attr("selected", "").Add(text)
}
