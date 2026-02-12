package dom

import (
	"github.com/tinywasm/fmt"
)

type SelectEl struct{ *Element }

func (s *SelectEl) AsElement() *Element { return s.Element }

// Semantic methods
func (s *SelectEl) Name(n string) *SelectEl { s.Element.Attr("name", n); return s }
func (s *SelectEl) Required() *SelectEl     { s.Element.Attr("required", ""); return s }
func (s *SelectEl) Disabled() *SelectEl     { s.Element.Attr("disabled", ""); return s }
func (s *SelectEl) Multiple() *SelectEl     { s.Element.Attr("multiple", ""); return s }

// Shadow methods
func (s *SelectEl) ID(id string) *SelectEl               { s.Element.ID(id); return s }
func (s *SelectEl) Class(c ...string) *SelectEl          { s.Element.Class(c...); return s }
func (s *SelectEl) Attr(k, v string) *SelectEl           { s.Element.Attr(k, v); return s }
func (s *SelectEl) On(t string, h func(Event)) *SelectEl { s.Element.On(t, h); return s }
func (s *SelectEl) Add(children ...any) *SelectEl        { s.Element.Add(children...); return s }

// Factory: name as first required arg, options as children
func Select(name string, children ...any) *SelectEl {
	return &SelectEl{&Element{tag: "select", children: children,
		attrs: []fmt.KeyValue{{Key: "name", Value: name}}}}
}
