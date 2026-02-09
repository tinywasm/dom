package html

import (
	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

// Tag creates a new Node.
func Tag(tag string, children ...any) dom.Node {
	n := dom.Node{
		Tag: tag,
	}
	for _, child := range children {
		switch v := child.(type) {
		case dom.Node:
			n.Children = append(n.Children, v)
		case string:
			n.Children = append(n.Children, v)
		case dom.Component:
			n.Children = append(n.Children, v)
		case func(dom.Event):
			// Default event is click if not specified?
			// No, better use OnClick helper.
		case attr:
			n.Attrs = append(n.Attrs, fmt.KeyValue{Key: v.key, Value: v.val})
		case event:
			n.Events = append(n.Events, dom.EventHandler{Name: v.name, Handler: v.handler})
		}
	}
	return n
}

type attr struct {
	key string
	val string
}

type event struct {
	name    string
	handler func(dom.Event)
}

// Common Tags
func Div(children ...any) dom.Node    { return Tag("div", children...) }
func Span(children ...any) dom.Node   { return Tag("span", children...) }
func Button(children ...any) dom.Node { return Tag("button", children...) }
func H1(children ...any) dom.Node     { return Tag("h1", children...) }
func H2(children ...any) dom.Node     { return Tag("h2", children...) }
func P(children ...any) dom.Node      { return Tag("p", children...) }
func Ul(children ...any) dom.Node     { return Tag("ul", children...) }
func Li(children ...any) dom.Node     { return Tag("li", children...) }

// Attributes
func ID(id string) attr       { return attr{"id", id} }
func Class(class string) attr { return attr{"class", class} }
func Attr(k, v string) attr   { return attr{k, v} }

// Events
func OnClick(h func(dom.Event)) event  { return event{"click", h} }
func OnInput(h func(dom.Event)) event  { return event{"input", h} }
func OnChange(h func(dom.Event)) event { return event{"change", h} }

// Text helper (though string works too)
func Text(s string) string { return s }
