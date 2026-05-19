//go:build wasm

package dom_test

// Reference mutation tests: verify that SetValue, SetAttr, RemoveAttr, and SetText
// mutate the DOM in-place without destroying event listeners registered via On().
//
// Fix summary:
//   - Bug 1 (SetValue): ref.SetValue("") resets input.value without re-rendering.
//   - Bug 2 (SetAttr/RemoveAttr): ref.SetAttr/RemoveAttr mutates attributes in-place.
//   - Bug 3 (SetText): ref.SetText(msg) sets textContent without nesting elements.

import (
	"strings"
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

// TestReference_SetValue_PreservesListeners verifies Bug 1 fix.
func TestReference_SetValue_PreservesListeners(t *testing.T) {
	SetupDOM(t)
	dom.Render("root", dom.Input("text").ID("f1-input"))
	ref, _ := dom.Get("f1-input")
	fired := false
	ref.On("input", func(e dom.Event) { fired = true })

	// Mutate via new API
	ref.SetValue("fixed")

	// Trigger event — listener MUST survive
	TriggerEvent("f1-input", "input", "fixed")

	if !fired {
		t.Error("FIX 1: SetValue destroyed the 'input' listener or didn't preserve it")
	}
	if ref.Value() != "fixed" {
		t.Errorf("Expected value 'fixed', got %q", ref.Value())
	}
}

// TestReference_SetAttr_PreservesListeners verifies Bug 2 fix.
func TestReference_SetAttr_PreservesListeners(t *testing.T) {
	SetupDOM(t)
	dom.Render("root", dom.Button("Click").ID("f2-btn"))
	ref, _ := dom.Get("f2-btn")
	clicked := false
	ref.On("click", func(e dom.Event) { clicked = true })

	// Mutate via new API
	ref.SetAttr("disabled", "")
	ref.RemoveAttr("disabled")

	// Trigger event — listener MUST survive
	TriggerEvent("f2-btn", "click", "")

	if !clicked {
		t.Error("FIX 2: SetAttr/RemoveAttr destroyed the 'click' listener")
	}
}

// TestReference_SetText_PreservesAttributes verifies Bug 3 fix.
func TestReference_SetText_PreservesAttributes(t *testing.T) {
	SetupDOM(t)
	dom.Render("root", dom.Span("").ID("f3-span").Attr("aria-live", "polite"))
	ref, _ := dom.Get("f3-span")

	ref.SetText("fixed text")

	rawEl := js.Global().Get("document").Call("getElementById", "f3-span")
	innerHTML := rawEl.Get("innerHTML").String()
	if strings.Contains(innerHTML, "<span") {
		t.Errorf("FIX 3: SetText nested a span. innerHTML=%q", innerHTML)
	}
	if innerHTML != "fixed text" {
		t.Errorf("Expected 'fixed text', got %q", innerHTML)
	}

	ariaLive := rawEl.Call("getAttribute", "aria-live").String()
	if ariaLive != "polite" {
		t.Errorf("FIX 3: aria-live lost after SetText, got %q", ariaLive)
	}
}

// TestReference_SetText_NoHTMLInjection verifies security.
func TestReference_SetText_NoHTMLInjection(t *testing.T) {
	SetupDOM(t)
	dom.Render("root", dom.Span("").ID("f4-span"))
	ref, _ := dom.Get("f4-span")

	ref.SetText("<img src=x onerror=alert(1)>")

	rawEl := js.Global().Get("document").Call("getElementById", "f4-span")
	innerHTML := rawEl.Get("innerHTML").String()
	if strings.Contains(innerHTML, "<img") {
		t.Errorf("SECURITY: SetText interpreted HTML. innerHTML=%q", innerHTML)
	}
}
