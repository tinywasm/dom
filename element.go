package dom

import (
	"github.com/tinywasm/fmt"
)

// Element represents a DOM element in the fluent Element API.
type Element struct {
	tag       string
	id        string
	key       string
	classes   []string
	attrs     []fmt.KeyValue
	events    []eventHandler
	bindings  []binding
	children  []any
	void      bool
	autofocus bool
}

type binding struct {
	kind   string // "text", "attr", "class", "attrbool", "value", "children"
	name   string // attr name or class name
	signal subscribable
	fnString func() string
	fnBool   func() bool
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

// Key sets a stable identity for keyed reconciliation in BindChildren.
func (b *Element) Key(key string) *Element {
	b.key = key
	return b
}

// Autofocus marks the element to be focused when it first appears.
func (b *Element) Autofocus() *Element {
	b.autofocus = true
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

// Child adds one or more elements or components as children.
func (b *Element) Child(c ...Component) *Element {
	for _, child := range c {
		if child != nil {
			b.children = append(b.children, child)
		}
	}
	return b
}

// Set applies multiple attributes or classes at once using KeyValue pairs.
func (b *Element) Set(kv ...fmt.KeyValue) *Element {
	for _, attr := range kv {
		switch attr.Key {
		case "class":
			b.Class(attr.Value)
		case "id":
			b.ID(attr.Value)
		default:
			b.Attr(attr.Key, attr.Value)
		}
	}
	return b
}

// Text adds a text node child.
func (b *Element) Text(text string) *Element {
	b.children = append(b.children, text)
	return b
}

// BindText links the element's textContent to a SignalString.
func (b *Element) BindText(s *SignalString) *Element {
	b.bindings = append(b.bindings, binding{kind: "text", signal: s})
	return b
}

// BindAttr links an attribute to a SignalString.
func (b *Element) BindAttr(name string, s *SignalString) *Element {
	b.bindings = append(b.bindings, binding{kind: "attr", name: name, signal: s})
	return b
}

// BindClass toggles a class based on a SignalBool.
func (b *Element) BindClass(class string, on *SignalBool) *Element {
	b.bindings = append(b.bindings, binding{kind: "class", name: class, signal: on})
	return b
}

// BindAttrBool toggles a boolean attribute (disabled, checked, etc.) based on a SignalBool.
func (b *Element) BindAttrBool(name string, on *SignalBool) *Element {
	b.bindings = append(b.bindings, binding{kind: "attrbool", name: name, signal: on})
	return b
}

// Bind provides two-way binding for <input> and <textarea>.
func (b *Element) Bind(s *SignalString) *Element {
	b.bindings = append(b.bindings, binding{kind: "value", signal: s})
	return b
}

// BindChildren links a container's children to a SignalNodes.
func (b *Element) BindChildren(s *SignalNodes) *Element {
	b.bindings = append(b.bindings, binding{kind: "children", signal: s})
	return b
}

// BindTextFunc links the element's textContent to a computed string.
func (b *Element) BindTextFunc(fn func() string) *Element {
	b.bindings = append(b.bindings, binding{kind: "text", fnString: fn})
	return b
}

// BindAttrFunc links an attribute to a computed string.
func (b *Element) BindAttrFunc(name string, fn func() string) *Element {
	b.bindings = append(b.bindings, binding{kind: "attr", name: name, fnString: fn})
	return b
}

// BindClassFunc toggles a class based on a computed boolean.
func (b *Element) BindClassFunc(class string, fn func() bool) *Element {
	b.bindings = append(b.bindings, binding{kind: "class", name: class, fnBool: fn})
	return b
}

// BindAttrBoolFunc toggles a boolean attribute based on a computed boolean.
func (b *Element) BindAttrBoolFunc(name string, fn func() bool) *Element {
	b.bindings = append(b.bindings, binding{kind: "attrbool", name: name, fnBool: fn})
	return b
}

// Render renders the element to the parent.
// This is a terminal operation.
func (b *Element) Render(parentID string) error {
	return Render(parentID, b)
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
	if el == nil {
		return ""
	}
	s := "<" + el.tag
	if el.id != "" {
		s += " id='" + el.id + "'"
	}

	classes := el.classes
	attrs := el.attrs
	textContent := ""
	hasTextContent := false

	for _, b := range el.bindings {
		switch b.kind {
		case "text":
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalString); ok {
					textContent = sig.Get()
				}
			} else if b.fnString != nil {
				textContent = b.fnString()
			}
			hasTextContent = true
		case "attr":
			val := ""
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalString); ok {
					val = sig.Get()
				}
			} else if b.fnString != nil {
				val = b.fnString()
			}
			found := false
			for i, attr := range attrs {
				if attr.Key == b.name {
					attrs[i].Value = val
					found = true
					break
				}
			}
			if !found {
				attrs = append(attrs, fmt.KeyValue{Key: b.name, Value: val})
			}
		case "class":
			on := false
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalBool); ok {
					on = sig.Get()
				}
			} else if b.fnBool != nil {
				on = b.fnBool()
			}
			if on {
				classes = append(classes, b.name)
			}
		case "attrbool":
			on := false
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalBool); ok {
					on = sig.Get()
				}
			} else if b.fnBool != nil {
				on = b.fnBool()
			}
			if on {
				attrs = append(attrs, fmt.KeyValue{Key: b.name, Value: ""})
			}
		case "value":
			val := ""
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalString); ok {
					val = sig.Get()
				}
			}
			attrs = append(attrs, fmt.KeyValue{Key: "value", Value: val})
		}
	}

	if len(classes) > 0 {
		s += " class='"
		for i, c := range classes {
			if i > 0 {
				s += " "
			}
			s += c
		}
		s += "'"
	}
	for _, attr := range attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
	s += ">"
	if el.void {
		return s
	}

	if hasTextContent {
		s += textContent
	} else {
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
	}
	s += "</" + el.tag + ">"
	return s
}

