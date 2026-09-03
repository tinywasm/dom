package dom

import (
	"github.com/tinywasm/fmt"
)

type childRenderer func(Component) string

type elementObserver func(*Element)

// serializeElement renders an Element tree into an HTML string.
// renderChild resolves Component children to HTML (SSR vs WASM lifecycle tracking).
// observer is an optional callback invoked for each Element in the tree (used by WASM to collect pending events).
func serializeElement(el *Element, renderChild childRenderer, observer ...elementObserver) string {
	if el == nil {
		return ""
	}
	beginPass()
	defer endPass()

	if (len(el.events) > 0 || len(el.bindings) > 0 || el.autofocus) && el.id == "" {
		el.id = generateID()
	}

	var obs elementObserver
	if len(observer) > 0 && observer[0] != nil {
		obs = observer[0]
		obs(el)
	}

	s := "<" + el.tag
	if el.id != "" {
		claimID(el.id, el.tag)
		s += " id='" + fmt.Convert(el.id).EscapeAttr() + "'"
	}

	classes := el.classes
	attrs := el.attrs
	textContent := ""
	hasTextContent := false
	var boundChildren []*Element

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
		case "state":
			on := false
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalBool); ok {
					on = sig.Get()
				}
			} else if b.fnBool != nil {
				on = b.fnBool()
			}
			if on {
				attrs = append(attrs, fmt.KeyValue{Key: b.state.Key(), Value: b.state.Value()})
			}
		case "value":
			val := ""
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalString); ok {
					val = sig.Get()
				}
			}
			attrs = append(attrs, fmt.KeyValue{Key: "value", Value: val})
		case "children":
			if sig, ok := b.signal.(*SignalNodes); ok {
				boundChildren = append(boundChildren, sig.Get()...)
			}
		}
	}

	if len(classes) > 0 {
		s += " class='"
		for i, c := range classes {
			if i > 0 {
				s += " "
			}
			s += fmt.Convert(c).EscapeAttr()
		}
		s += "'"
	}
	for _, attr := range attrs {
		s += " " + fmt.Convert(attr.Key).EscapeAttr() + "='" + fmt.Convert(attr.Value).EscapeAttr() + "'"
	}
	s += ">"
	if el.void {
		return s
	}

	if hasTextContent {
		s += fmt.Convert(textContent).EscapeHTML()
	} else {
		for _, node := range boundChildren {
			s += serializeElement(node, renderChild, obs)
		}
		for _, child := range el.children {
			switch v := child.(type) {
			case *Element:
				s += serializeElement(v, renderChild, obs)
			case TrustedHTML:
				s += string(v)
			case string:
				s += fmt.Convert(v).EscapeHTML()
			case Component:
				s += renderChild(v)
			default:
				s += fmt.Convert(fmt.Sprint(v)).EscapeHTML()
			}
		}
	}
	s += "</" + el.tag + ">"
	return s
}
