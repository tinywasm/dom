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
