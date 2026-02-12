package dom

type FormEl struct{ *Element }

func (f *FormEl) AsElement() *Element { return f.Element }

// Semantic methods
func (f *FormEl) Action(url string) *FormEl       { f.Element.Attr("action", url); return f }
func (f *FormEl) Method(m string) *FormEl         { f.Element.Attr("method", m); return f }
func (f *FormEl) NoValidate() *FormEl             { f.Element.Attr("novalidate", ""); return f }
func (f *FormEl) OnSubmit(fn func(Event)) *FormEl { f.Element.On("submit", fn); return f }

// Shadow methods
func (f *FormEl) ID(id string) *FormEl               { f.Element.ID(id); return f }
func (f *FormEl) Class(c ...string) *FormEl          { f.Element.Class(c...); return f }
func (f *FormEl) Attr(k, v string) *FormEl           { f.Element.Attr(k, v); return f }
func (f *FormEl) On(t string, h func(Event)) *FormEl { f.Element.On(t, h); return f }
func (f *FormEl) Add(children ...any) *FormEl        { f.Element.Add(children...); return f }

// Factory
func Form(children ...any) *FormEl {
	return &FormEl{&Element{tag: "form", children: children}}
}
