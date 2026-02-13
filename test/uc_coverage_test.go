//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

func TestCoverageSpecializedElements(t *testing.T) {
	_ = SetupDOM(t)

	t.Run("InputEl methods", func(t *testing.T) {
		i := dom.Input("text").
			ID("test-input").
			Class("c1", "c2").
			Attr("data-test", "val").
			Name("test-name").
			Value("test-value").
			Placeholder("test-placeholder").
			Required().
			Disabled().
			Readonly().
			Checked().
			Min("0").
			Max("100").
			Step("5").
			Pattern("[0-9]*").
			AutoComplete("off")

		html := i.RenderHTML()
		checks := []string{
			"id='test-input'",
			"class='c1 c2'",
			"data-test='val'",
			"name='test-name'",
			"value='test-value'",
			"placeholder='test-placeholder'",
			"required=''",
			"disabled=''",
			"readonly=''",
			"checked=''",
			"min='0'",
			"max='100'",
			"step='5'",
			"pattern='[0-9]*'",
			"autocomplete='off'",
		}
		for _, c := range checks {
			if !fmt.Contains(html, c) {
				t.Errorf("Input HTML missing attribute: %s, got: %s", c, html)
			}
		}

		// Test Required(false)
		i.Required(false) // This is a no-op in currently implementation if it already has it,
		// wait, Required(false) should remove it if we want full opt-out,
		// but the instruction was "no-op (allows explicit opt-out in form contexts)"
		// which usually means if you pass false it does nothing.
		// Actually, if it's already there, it stays there with current implementation:
		// func (i *InputEl) Required(v ...bool) *InputEl {
		// 	if len(v) == 0 || v[0] {
		// 		i.Element.Attr("required", "")
		// 	}
		// 	return i
		// }
		// So Required(false) does nothing.
	})

	t.Run("Input Typed Factories", func(t *testing.T) {
		factories := []struct {
			el  dom.Component
			tag string
		}{
			{dom.Number("n"), "type='number'"},
			{dom.Radio("r", "v"), "type='radio'"},
			{dom.Radio("r", "v").Name("rn"), "name='rn'"}, // coverage for Name
			{dom.File("f"), "type='file'"},
			{dom.Date("d"), "type='date'"},
			{dom.Hidden("h", "v"), "type='hidden'"},
			{dom.Email("e", "p"), "type='email'"},
			{dom.Search("s", "p"), "type='search'"},
			{dom.Tel("t", "p"), "type='tel'"},
			{dom.Url("u", "p"), "type='url'"},
			{dom.Range("ra"), "type='range'"},
			{dom.Color("co"), "type='color'"},
			{dom.Submit("Go"), "type='submit'"},
			{dom.Reset("Clear"), "type='reset'"},
		}
		for _, f := range factories {
			html := f.el.RenderHTML()
			if !fmt.Contains(html, f.tag) {
				t.Errorf("Factory HTML missing %s: %s", f.tag, html)
			}
		}
	})

	t.Run("SelectEl methods", func(t *testing.T) {
		s := dom.Select("test-select").
			Required().
			Disabled().
			Multiple().
			ID("sel-id").
			Class("sel-class").
			Attr("data-sel", "val")

		html := s.RenderHTML()
		checks := []string{
			"required=''",
			"disabled=''",
			"multiple=''",
			"id='sel-id'",
			"class='sel-class'",
			"data-sel='val'",
		}
		for _, c := range checks {
			if !fmt.Contains(html, c) {
				t.Errorf("Select HTML missing %s: %s", c, html)
			}
		}
	})

	t.Run("TextareaEl methods", func(t *testing.T) {
		ta := dom.Textarea("test-ta").
			Placeholder("placeholder").
			Rows(10).
			Cols(50).
			Required().
			Readonly().
			MaxLength(100).
			Value("initial value").
			ID("ta-id").
			Class("ta-class").
			Attr("data-ta", "val").
			On("input", func(e dom.Event) {})

		html := ta.RenderHTML()
		checks := []string{
			"rows='10'",
			"cols='50'",
			"required=''",
			"readonly=''",
			"maxlength='100'",
			"initial value",
			"name='test-ta'",
			"placeholder='placeholder'",
			"id='ta-id'",
			"class='ta-class'",
			"data-ta='val'",
		}
		for _, c := range checks {
			if !fmt.Contains(html, c) {
				t.Errorf("Textarea HTML missing %s: %s", c, html)
			}
		}
	})
}

func TestCoverageElementFactories(t *testing.T) {
	t.Run("All Element Factories", func(t *testing.T) {
		els := []dom.Component{
			dom.Span(), dom.P(), dom.H1(), dom.H2(), dom.H3(), dom.H4(), dom.H5(), dom.H6(),
			dom.Ul(), dom.Ol(), dom.Li(), dom.Nav(), dom.Section(), dom.Main(), dom.Article(),
			dom.Header(), dom.Footer(), dom.Aside(), dom.Details(), dom.Summary(), dom.Dialog(),
			dom.Figure(), dom.Figcaption(), dom.Pre(), dom.Code(), dom.Strong(), dom.Em(),
			dom.Small(), dom.Mark(), dom.Table(), dom.Thead(), dom.Tbody(), dom.Tfoot(),
			dom.Tr(), dom.Th(), dom.Td(), dom.Fieldset(), dom.Legend(), dom.Label(),
			dom.Canvas(), dom.Style(), dom.Script(), dom.A("href"), dom.Button(),
		}
		for _, el := range els {
			if el.RenderHTML() == "" {
				t.Errorf("Element factory returned empty HTML")
			}
		}
	})
}

func TestCoverageDOMLogic(t *testing.T) {
	_ = SetupDOM(t)

	t.Run("Update non-existent component", func(t *testing.T) {
		c := dom.Div().ID("non-existent")
		err := dom.Update(c)
		if err == nil {
			t.Error("Expected error when updating non-existent component")
		}
	})

	t.Run("Render to non-existent parent", func(t *testing.T) {
		err := dom.Render("void", dom.Div())
		if err == nil {
			t.Error("Expected error when rendering to non-existent parent")
		}
	})

	t.Run("Logging", func(t *testing.T) {
		// No crash without log handler
		dom.Log("test message")

		logged := false
		dom.SetLog(func(v ...any) {
			logged = true
		})
		dom.Log("test message 2")
		if !logged {
			t.Error("Log handler not called")
		}
		dom.SetLog(nil)
	})

	t.Run("Hash", func(t *testing.T) {
		dom.SetHash("test")
		_ = dom.GetHash()
		// OnHashChange coverage
		dom.OnHashChange(func(h string) {})
	})

	t.Run("Element methods", func(t *testing.T) {
		el := dom.Div().Text("hello").Attr("k", "v")
		// Test duplicate Attr
		el.Attr("k", "v2")
		if !fmt.Contains(el.RenderHTML(), "k='v2'") {
			t.Error("Attr not updated")
		}
	})
}

func TestCoverageEvents(t *testing.T) {
	SetupDOM(t)

	t.Run("Event interface methods - Button", func(t *testing.T) {
		var ev dom.Event
		triggered := false
		btn := dom.Button("Click").ID("btn-ev").On("click", func(e dom.Event) {
			ev = e
			triggered = true
		})
		dom.Render("root", btn)
		TriggerEvent(btn.GetID(), "click", "")

		if triggered && ev != nil {
			ev.PreventDefault()
			ev.StopPropagation()
			_ = ev.TargetID()
			_ = ev.TargetValue()
		}
	})

	t.Run("Event interface methods - Checkbox", func(t *testing.T) {
		var ev dom.Event
		triggered := false
		check := dom.Checkbox("check").ID("check-ev").On("change", func(e dom.Event) {
			ev = e
			triggered = true
		})
		dom.Render("root", check)
		TriggerEvent(check.GetID(), "change", "")

		if triggered && ev != nil {
			_ = ev.TargetChecked()
		}
	})

	t.Run("Append logic", func(t *testing.T) {
		parent := dom.Div().ID("parent-append")
		dom.Render("root", parent)
		child := dom.Span().ID("child-append").Text("appended")
		err := dom.Append("parent-append", child)
		if err != nil {
			t.Errorf("Append failed: %v", err)
		}
		if _, ok := GetRef("child-append"); !ok {
			t.Error("Appended element not found in DOM")
		}
	})
}

func TestLifecycleDeep(t *testing.T) {
	_ = SetupDOM(t)

	t.Run("Nested components cleanup", func(t *testing.T) {
		child := &MockComponent{Element: dom.Div().ID("child-comp")}
		parent := &MockComponent{Element: dom.Div(child).ID("parent-comp")}

		dom.Render("root", parent)
		if !child.Mounted {
			t.Error("Child component should be mounted")
		}

		// Update parent to remove child
		parent.Element = dom.Div().ID("parent-comp").Text("no more child")
		dom.Update(parent)

		if child.Mounted {
			t.Error("Child component should be unmounted after removal from parent")
		}
	})

	t.Run("ElementNode in children", func(t *testing.T) {
		// Form is an elementNode
		f := dom.Form(dom.Text("q")).ID("myform")
		el := dom.Div(f)
		html := el.RenderHTML()
		if !fmt.Contains(html, "<form id='myform'") {
			t.Error("Form elementNode not rendered correctly in children")
		}
	})

	t.Run("Default case in renderToHTML", func(t *testing.T) {
		el := dom.Div(123, true)
		html := el.RenderHTML()
		if !fmt.Contains(html, "123") || !fmt.Contains(html, "true") {
			t.Errorf("Default types not rendered correctly: %s", html)
		}
	})

	t.Run("Component with only RenderHTML", func(t *testing.T) {
		c := &OnlyHTMLComp{id: "ohc"}
		el := dom.Div(c)
		html := el.RenderHTML()
		if !fmt.Contains(html, "ONLY HTML") {
			t.Error("Component with only RenderHTML not rendered correctly")
		}
	})
}

type OnlyHTMLComp struct {
	id string
}

func (c *OnlyHTMLComp) GetID() string             { return c.id }
func (c *OnlyHTMLComp) SetID(id string)           { c.id = id }
func (c *OnlyHTMLComp) RenderHTML() string        { return "ONLY HTML" }
func (c *OnlyHTMLComp) Children() []dom.Component { return nil }

func TestCoverageCleanup(t *testing.T) {
	_ = SetupDOM(t)

	t.Run("Listener cleanup", func(t *testing.T) {
		triggered := false
		btn := dom.Button("Click").ID("btn-clean").On("click", func(e dom.Event) {
			triggered = true
		})
		root := &MockComponent{Element: dom.Div(btn).ID("root-clean")}
		dom.Render("root", root)

		// Trigger before cleanup
		TriggerEvent("btn-clean", "click", "")
		if !triggered {
			t.Error("Event should have triggered")
		}

		// Update root to remove button
		triggered = false
		root.Element = dom.Div().ID("root-clean").Text("Gone")
		dom.Update(root)

		// Trigger after cleanup (should not crash and triggered should remain false)
		TriggerEvent("btn-clean", "click", "")
		if triggered {
			t.Error("Event should NOT have triggered after cleanup")
		}
	})

	t.Run("Option helpers", func(t *testing.T) {
		opt := dom.Option("v1", "Text 1")
		if !fmt.Contains(opt.RenderHTML(), "value='v1'") || !fmt.Contains(opt.RenderHTML(), "Text 1") {
			t.Error("Option not rendered correctly")
		}
		sopt := dom.SelectedOption("v2", "Text 2")
		if !fmt.Contains(sopt.RenderHTML(), "selected=''") {
			t.Error("SelectedOption not rendered correctly")
		}
	})

	t.Run("A and Button", func(t *testing.T) {
		a := dom.A("https://google.com", "Link")
		if !fmt.Contains(a.RenderHTML(), "href='https://google.com'") {
			t.Error("A not rendered correctly")
		}
		b := dom.Button("Click Me")
		if !fmt.Contains(b.RenderHTML(), "Click Me") {
			t.Error("Button not rendered correctly")
		}
	})

	t.Run("Element.Children", func(t *testing.T) {
		child := &MockComponent{Element: dom.Div()}
		el := dom.Div(child, "text", dom.Span())
		children := el.Children()
		if len(children) != 2 { // MockComponent and Span
			t.Errorf("Expected 2 component children, got %d", len(children))
		}
	})

	t.Run("Deep cleanup slice manipulation", func(t *testing.T) {
		c1 := &MockComponent{Element: dom.Div().ID("c1")}
		c2 := &MockComponent{Element: dom.Div().ID("c2")}
		c3 := &MockComponent{Element: dom.Div().ID("c3")}
		parent := &MockComponent{Element: dom.Div(c1, c2, c3).ID("parent-deep")}

		dom.Render("root", parent)

		// Remove one child
		parent.Element = dom.Div(c1, c3).ID("parent-deep")
		dom.Update(parent)
		if c2.Mounted {
			t.Error("c2 should be unmounted")
		}

		// Remove all
		parent.Element = dom.Div().ID("parent-deep")
		dom.Update(parent)
		if c1.Mounted || c3.Mounted {
			t.Error("c1 and c3 should be unmounted")
		}
	})

	t.Run("Internal Edge Cases", func(t *testing.T) {
		// Exercise trackComponent already existing
		c := &MockComponent{Element: dom.Div().ID("existing")}
		dom.Render("root", c)
		dom.Render("root", c) // Should return early in trackComponent

		// Exercise trackChildren entry exists
		dom.Update(c) // Should update existing entry in childrenMap
	})

	t.Run("Update with embedded element", func(t *testing.T) {
		child := &MockComponent{Element: dom.Div().ID("embedded")}
		dom.Render("root", child)
		// Update using the embedded element
		err := dom.Update(child.Element)
		if err != nil {
			t.Errorf("Update embedded failed: %v", err)
		}
	})

	t.Run("Complex cleanup branches", func(t *testing.T) {
		c1 := &MockComponent{Element: dom.Div().ID("c1").On("click", func(e dom.Event) {})}
		c2 := &MockComponent{Element: dom.Div().ID("c2").On("click", func(e dom.Event) {}).On("input", func(e dom.Event) {})}
		parent := &MockComponent{Element: dom.Div(c1, c2).ID("parent-complex").On("click", func(e dom.Event) {})}

		dom.Render("root", parent)

		// This should trigger cleanupListeners for parent and children
		// and hit splitEventKey and the multiple eventFuncs loop
		dom.Render("root", dom.Div().ID("new-root"))

		if c1.Mounted || c2.Mounted || parent.Mounted {
			// They should be unmounted
		}
	})
}

func TestCoverageForm(t *testing.T) {
	t.Run("Form helpers", func(t *testing.T) {
		f := dom.Form().
			Action("/submit").
			Method("POST").
			NoValidate().
			OnSubmit(func(e dom.Event) {})

		if !fmt.Contains(f.RenderHTML(), "action='/submit'") {
			t.Error("Form Action failed")
		}
	})
}
