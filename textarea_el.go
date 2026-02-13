package dom

import (
	"github.com/tinywasm/fmt"
)

type TextareaEl struct{ *Element }

func (t *TextareaEl) AsElement() *Element { return t.Element }

// Semantic methods
func (t *TextareaEl) Name(n string) *TextareaEl        { t.Element.Attr("name", n); return t }
func (t *TextareaEl) Rows(n int) *TextareaEl           { t.Element.Attr("rows", fmt.Sprint(n)); return t }
func (t *TextareaEl) Cols(n int) *TextareaEl           { t.Element.Attr("cols", fmt.Sprint(n)); return t }
func (t *TextareaEl) Placeholder(p string) *TextareaEl { t.Element.Attr("placeholder", p); return t }
func (t *TextareaEl) Required(v ...bool) *TextareaEl {
	if len(v) == 0 || v[0] {
		t.Element.Attr("required", "")
	}
	return t
}
func (t *TextareaEl) Readonly() *TextareaEl { t.Element.Attr("readonly", ""); return t }
func (t *TextareaEl) MaxLength(n int) *TextareaEl {
	t.Element.Attr("maxlength", fmt.Sprint(n))
	return t
}
func (t *TextareaEl) Value(v string) *TextareaEl { t.Element.Text(v); return t }

// Shadow methods
func (t *TextareaEl) ID(id string) *TextareaEl                { t.Element.ID(id); return t }
func (t *TextareaEl) Class(c ...string) *TextareaEl           { t.Element.Class(c...); return t }
func (t *TextareaEl) Attr(k, v string) *TextareaEl            { t.Element.Attr(k, v); return t }
func (t *TextareaEl) On(ev string, h func(Event)) *TextareaEl { t.Element.On(ev, h); return t }

// Factory: (name, placeholder) — defaults to rows="3"
func Textarea(name string, placeholder ...string) *TextareaEl {
	el := &Element{tag: "textarea",
		attrs: []fmt.KeyValue{
			{Key: "name", Value: name},
			{Key: "rows", Value: "3"},
		}}
	if len(placeholder) > 0 {
		el.attrs = append(el.attrs, fmt.KeyValue{Key: "placeholder", Value: placeholder[0]})
	}
	return &TextareaEl{el}
}
