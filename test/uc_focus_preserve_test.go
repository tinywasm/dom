//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

// FocusUpdater is a component with a text input that calls Update() on every
// keystroke — exactly the selectsearch pattern.
type FocusUpdater struct {
	dom.Element
	filterTerm string
}

func (c *FocusUpdater) Render() *dom.Element {
	return dom.Div(
		dom.Input("text").ID(c.GetID()+"-input").Attr("value", c.filterTerm),
	)
}

func (c *FocusUpdater) OnMount() {
	id := c.GetID()
	if el, ok := dom.Get(id + "-input"); ok {
		el.On("input", func(e dom.Event) {
			c.filterTerm = e.TargetValue()
			c.Update()
		})
	}
}

// TestUpdate_PreservesActiveElementFocus verifies that calling Update() while an
// input element is focused does not steal focus from it.
//
// This test FAILS before the fix in dom_frontend.go because outerHTML replacement
// destroys the active element and moves focus to document.body.
func TestUpdate_PreservesActiveElementFocus(t *testing.T) {
	SetupDOM(t)

	c := &FocusUpdater{}
	if err := dom.Render("root", c); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	id := c.GetID()
	doc := js.Global().Get("document")

	// Focus the input and set a value to simulate the user typing.
	inputEl := doc.Call("getElementById", id+"-input")
	if inputEl.IsNull() {
		t.Fatal("input element not found")
	}
	inputEl.Call("focus")
	inputEl.Set("value", "hi")
	// Place cursor at the end (position 2).
	inputEl.Call("setSelectionRange", 2, 2)

	// Dispatch an input event — this triggers OnMount's handler → c.Update().
	event := js.Global().Get("InputEvent").New("input", map[string]interface{}{
		"bubbles": true,
	})
	inputEl.Call("dispatchEvent", event)

	// After Update(), the active element must still be the input.
	activeID := doc.Get("activeElement").Get("id").String()
	if activeID != id+"-input" {
		t.Errorf("focus lost after Update(): activeElement.id=%q, want %q", activeID, id+"-input")
	}
}
