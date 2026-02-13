package dom

import (
	"github.com/tinywasm/fmt"
)

type InputEl struct{ *Element }

func (i *InputEl) AsElement() *Element { return i.Element }

// Semantic methods — all return *InputEl
func (i *InputEl) Name(n string) *InputEl        { i.Element.Attr("name", n); return i }
func (i *InputEl) Value(v string) *InputEl       { i.Element.Attr("value", v); return i }
func (i *InputEl) Placeholder(p string) *InputEl { i.Element.Attr("placeholder", p); return i }
func (i *InputEl) Required(v ...bool) *InputEl {
	if len(v) == 0 || v[0] {
		i.Element.Attr("required", "")
	}
	return i
}
func (i *InputEl) Disabled() *InputEl             { i.Element.Attr("disabled", ""); return i }
func (i *InputEl) Readonly() *InputEl             { i.Element.Attr("readonly", ""); return i }
func (i *InputEl) Checked() *InputEl              { i.Element.Attr("checked", ""); return i }
func (i *InputEl) Min(v string) *InputEl          { i.Element.Attr("min", v); return i }
func (i *InputEl) Max(v string) *InputEl          { i.Element.Attr("max", v); return i }
func (i *InputEl) Step(v string) *InputEl         { i.Element.Attr("step", v); return i }
func (i *InputEl) Pattern(p string) *InputEl      { i.Element.Attr("pattern", p); return i }
func (i *InputEl) AutoComplete(v string) *InputEl { i.Element.Attr("autocomplete", v); return i }

// Shadow methods — preserve *InputEl chain
func (i *InputEl) ID(id string) *InputEl               { i.Element.ID(id); return i }
func (i *InputEl) Class(c ...string) *InputEl          { i.Element.Class(c...); return i }
func (i *InputEl) Attr(k, v string) *InputEl           { i.Element.Attr(k, v); return i }
func (i *InputEl) On(t string, h func(Event)) *InputEl { i.Element.On(t, h); return i }

// Base factory
func Input(inputType string) *InputEl {
	return &InputEl{&Element{tag: "input", void: true,
		attrs: []fmt.KeyValue{{Key: "type", Value: inputType}}}}
}

// Typed factories — (name string, placeholder ...string) where applicable
func Text(name string, placeholder ...string) *InputEl {
	i := Input("text").Name(name)
	if len(placeholder) > 0 {
		i.Placeholder(placeholder[0])
	}
	return i
}

func Email(name string, placeholder ...string) *InputEl {
	i := Input("email").Name(name)
	if len(placeholder) > 0 {
		i.Placeholder(placeholder[0])
	}
	return i
}

func Password(name string) *InputEl {
	return Input("password").Name(name)
}

func Number(name string) *InputEl {
	return Input("number").Name(name)
}

func Checkbox(name string, value ...string) *InputEl {
	i := Input("checkbox").Name(name)
	if len(value) > 0 {
		i.Value(value[0])
	}
	return i
}

func Radio(name, value string) *InputEl {
	return Input("radio").Name(name).Value(value)
}

func File(name string) *InputEl {
	return Input("file").Name(name)
}

func Date(name string) *InputEl {
	return Input("date").Name(name)
}

func Hidden(name, value string) *InputEl {
	return Input("hidden").Name(name).Value(value)
}

func Search(name string, placeholder ...string) *InputEl {
	i := Input("search").Name(name)
	if len(placeholder) > 0 {
		i.Placeholder(placeholder[0])
	}
	return i
}

func Tel(name string, placeholder ...string) *InputEl {
	i := Input("tel").Name(name)
	if len(placeholder) > 0 {
		i.Placeholder(placeholder[0])
	}
	return i
}

func Url(name string, placeholder ...string) *InputEl {
	i := Input("url").Name(name)
	if len(placeholder) > 0 {
		i.Placeholder(placeholder[0])
	}
	return i
}

func Range(name string) *InputEl {
	return Input("range").Name(name)
}

func Color(name string) *InputEl {
	return Input("color").Name(name)
}

func Submit(value ...string) *InputEl {
	i := Input("submit")
	if len(value) > 0 {
		i.Value(value[0])
	}
	return i
}

func Reset(value ...string) *InputEl {
	i := Input("reset")
	if len(value) > 0 {
		i.Value(value[0])
	}
	return i
}
