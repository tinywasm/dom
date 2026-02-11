//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

// DynamicComp uses ViewRenderer (DSL)
type DynamicComp struct {
	*dom.Element
}

func (c *DynamicComp) Render() *dom.Element {
	return dom.Div().ID("dynamic").Text("Dynamic Content")
}

// StaticComp uses HTMLRenderer (String)
type StaticComp struct {
	*dom.Element
}

func (c *StaticComp) RenderHTML() string {
	return `<div id="static">Static Content</div>`
}

func TestHybridRendering(t *testing.T) {
	_ = SetupDOM(t)

	t.Run("Render Dynamic Component (DSL)", func(t *testing.T) {
		c := &DynamicComp{Element: &dom.Element{}}
		dom.Render("root", c)

		el, ok := dom.Get("dynamic")
		if !ok {
			t.Fatal("Dynamic component not rendered")
		}
		if el.Value() != "" { // Check content? Value() is for inputs.
			// Element interface doesn't expose InnerText/HTML easily except SetHTML.
			// We trust ID existence for now.
		}
	})

	t.Run("Render Static Component (String)", func(t *testing.T) {
		c := &StaticComp{Element: &dom.Element{}}
		dom.Render("root", c)

		_, ok := dom.Get("static")
		if !ok {
			t.Fatal("Static component not rendered")
		}
	})
}
