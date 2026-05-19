//go:build wasm

package dom_test

// BUG REPRO: Reference interface is read-only — mutating DOM state without re-render is impossible.
//
// Bug 1 — No SetValue: the only way to reset an input value is dom.Render on the parent,
//          which calls cleanupChildren() and destroys all event listeners registered via On().
//
// Bug 2 — No SetAttr/RemoveAttr: disabling a button requires dom.Render, same destruction.
//
// Bug 3 — No SetText: updating an error span via dom.Render("errID", dom.Span(msg)) nests
//          a new <span> inside the existing one, accumulating on every update.
//
// All three tests compile and FAIL with the current code, proving the bugs exist.

import (
	"strings"
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

// TestBug_RenderDestroysListeners_NoSetValue reproduces Bug 1.
//
// After the ONLY available workaround for resetting an input (dom.Render on parent),
// the "input" event listener is destroyed. The input becomes unresponsive.
func TestBug_RenderDestroysListeners_NoSetValue(t *testing.T) {
	SetupDOM(t)

	// 1. Render an input inside root
	dom.Render("root", dom.Input("text").ID("ri-input"))

	// 2. Register listener via dom.Reference.On (the only registration API)
	ref, ok := dom.Get("ri-input")
	if !ok {
		t.Fatal("ri-input not found")
	}
	fired := false
	ref.On("input", func(e dom.Event) { fired = true })

	// 3. Verify listener works before the workaround
	TriggerEvent("ri-input", "input", "hello")
	if !fired {
		t.Fatal("pre-condition failed: listener should fire before re-render")
	}
	fired = false

	// 4. Reset input: no SetValue exists, must re-render parent
	dom.Render("root", dom.Input("text").ID("ri-input"))

	// 5. Trigger event — listener was destroyed by cleanupChildren inside Render
	TriggerEvent("ri-input", "input", "")

	// This assertion FAILS before the fix (fired == false after re-render)
	if !fired {
		t.Error("BUG 1: dom.Render on parent destroyed the 'input' listener — " +
			"Reference needs SetValue(string) to update element.value without re-rendering")
	}
}

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

// TestBug_RenderDestroysListeners_NoSetAttr reproduces Bug 2.
//
// To disable a button (loading state on submit), dom.Render is the only option.
// It destroys the submit listener, so the form can never be submitted again.
func TestBug_RenderDestroysListeners_NoSetAttr(t *testing.T) {
	SetupDOM(t)

	// 1. Render a div wrapping a button (dom.Form does not exist — use Div as container)
	dom.Render("root", dom.Div(
		dom.Button("Enviar").Attr("type", "submit").ID("sa-btn"),
	).ID("sa-form"))

	// 2. Register click listener on the button
	ref, ok := dom.Get("sa-btn")
	if !ok {
		t.Fatal("sa-btn not found")
	}
	clicked := false
	ref.On("click", func(e dom.Event) { clicked = true })

	// 3. Verify listener works before
	TriggerEvent("sa-btn", "click", "")
	if !clicked {
		t.Fatal("pre-condition failed: click listener should fire before re-render")
	}
	clicked = false

	// 4. "Disable" button: no SetAttr exists, must re-render parent
	dom.Render("root", dom.Div(
		dom.Button("Enviando...").Attr("type", "submit").Attr("disabled", "true").ID("sa-btn"),
	).ID("sa-form"))

	// 5. Re-enable: no RemoveAttr exists, must re-render parent again
	dom.Render("root", dom.Div(
		dom.Button("Enviar").Attr("type", "submit").ID("sa-btn"),
	).ID("sa-form"))

	// 6. Try to click again — listener was destroyed by the two re-renders
	TriggerEvent("sa-btn", "click", "")

	// This assertion FAILS before the fix
	if !clicked {
		t.Error("BUG 2: dom.Render to toggle button disabled state destroyed the click listener — " +
			"Reference needs SetAttr(key,value) and RemoveAttr(key) to mutate attributes without re-rendering")
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

// TestBug_RenderNestsSpans_NoSetText reproduces Bug 3.
//
// Updating an error span's text via dom.Render nests a new <span> inside the existing
// one on every call. After two updates the DOM has three nested spans.
func TestBug_RenderNestsSpans_NoSetText(t *testing.T) {
	SetupDOM(t)

	// 1. Render error span (as SSR would produce it)
	dom.Render("root", dom.Span("").
		ID("ns-field.error").
		Class("tw-field-error").
		Attr("aria-live", "polite"),
	)

	rawEl := js.Global().Get("document").Call("getElementById", "ns-field.error")
	if rawEl.IsNull() {
		t.Fatal("ns-field.error not found")
	}

	// 2. "Update" text: no SetText exists, must re-render into the span as parent
	dom.Render("ns-field.error", dom.Span("campo requerido").Class("tw-field-error--visible"))

	innerHTML := rawEl.Get("innerHTML").String()

	// Should be plain text or a single element — not nested <span> inside the outer <span>
	if strings.Contains(innerHTML, "<span") {
		t.Errorf("BUG 3: dom.Render('ns-field.error', dom.Span(...)) nested a <span> inside "+
			"the existing span. innerHTML=%q — "+
			"Reference needs SetText(string) to set textContent directly without nesting", innerHTML)
	}

	// 3. aria-live must survive the update
	ariaLive := rawEl.Call("getAttribute", "aria-live").String()
	if ariaLive != "polite" {
		t.Errorf("BUG 3: aria-live attribute lost after dom.Render update, got %q", ariaLive)
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
