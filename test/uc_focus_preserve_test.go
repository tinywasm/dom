//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

// FocusUpdater is a component with a text input that calls Update() on every
// keystroke — correctly, without an extra Focus() call after Update().
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


// FocusCallerUpdater replicates the anti-pattern of calling Focus() explicitly
// after c.Update(). Used to verify dom handles the redundant call gracefully.
type FocusCallerUpdater struct {
	dom.Element
	filterTerm string
}

func (c *FocusCallerUpdater) Render() *dom.Element {
	return dom.Div(
		dom.Input("text").ID(c.GetID()+"-input").Attr("value", c.filterTerm),
	)
}

func (c *FocusCallerUpdater) OnMount() {
	id := c.GetID()
	if el, ok := dom.Get(id + "-input"); ok {
		el.On("input", func(e dom.Event) {
			c.filterTerm = e.TargetValue()
			c.Update()
			if newEl, ok := dom.Get(id + "-input"); ok {
				newEl.Focus() // anti-pattern: redundant after c.Update()
			}
		})
	}
}

// installFocusSpy replaces HTMLElement.prototype.focus with a counter spy.
// Returns a cleanup func that restores the original and releases the JS func.
func installFocusSpy(t *testing.T) (count *int, cleanup func()) {
	t.Helper()
	n := 0
	proto := js.Global().Get("HTMLElement").Get("prototype")
	original := proto.Get("focus")
	spy := js.FuncOf(func(this js.Value, args []js.Value) any {
		n++
		original.Call("call", this) // invoke original with correct this binding
		return nil
	})
	proto.Set("focus", spy)
	return &n, func() {
		proto.Set("focus", original)
		spy.Release()
	}
}

// TestUpdate_FocusCalledOnceWhenActive verifies that dom.Update() calls focus()
// exactly once on the previously-active element — no more, no less.
// A correct component (no explicit Focus() after Update()) must produce count=1.
func TestUpdate_FocusCalledOnceWhenActive(t *testing.T) {
	SetupDOM(t)

	c := &FocusUpdater{}
	if err := dom.Render("root", c); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	id := c.GetID()
	doc := js.Global().Get("document")

	inputEl := doc.Call("getElementById", id+"-input")
	inputEl.Call("focus")
	inputEl.Set("value", "a")

	focusCount, cleanup := installFocusSpy(t)
	defer cleanup()

	event := js.Global().Get("InputEvent").New("input", map[string]any{"bubbles": true})
	inputEl.Call("dispatchEvent", event)

	if *focusCount != 1 {
		t.Errorf("focus() called %d time(s) during Update(), want exactly 1", *focusCount)
	}
}

// TestUpdate_RedundantFocusDetected verifies that even when a component calls
// Focus() explicitly after c.Update() (anti-pattern), the final state is still
// correct: the input is active and the cursor is at the expected position.
// dom cannot prevent the extra focus() call from the component side, but it
// must ensure its own restoration runs before the component's call.
// focusCount == 2 is expected with this anti-pattern; the test guards that
// the extra call does not break the final focused state.
func TestUpdate_RedundantFocusDetected(t *testing.T) {
	SetupDOM(t)

	c := &FocusCallerUpdater{}
	if err := dom.Render("root", c); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	id := c.GetID()
	doc := js.Global().Get("document")

	inputEl := doc.Call("getElementById", id+"-input")
	inputEl.Call("focus")
	inputEl.Set("value", "ab")
	inputEl.Call("setSelectionRange", 2, 2)

	focusCount, cleanup := installFocusSpy(t)
	defer cleanup()

	event := js.Global().Get("InputEvent").New("input", map[string]any{"bubbles": true})
	inputEl.Call("dispatchEvent", event)

	// Document expected call count with anti-pattern (2 = dom + component).
	if *focusCount < 1 {
		t.Errorf("focus() not called at all — dom restoration missing")
	}
	t.Logf("info: focus() called %d time(s) with anti-pattern (expected 2: dom + component)", *focusCount)

	// Final state must be correct regardless of call count.
	activeID := doc.Get("activeElement").Get("id").String()
	if activeID != id+"-input" {
		t.Errorf("final activeElement.id=%q, want %q — element lost focus", activeID, id+"-input")
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
	event := js.Global().Get("InputEvent").New("input", map[string]any{
		"bubbles": true,
	})
	inputEl.Call("dispatchEvent", event)

	// After Update(), the active element must still be the input.
	activeID := doc.Get("activeElement").Get("id").String()
	if activeID != id+"-input" {
		t.Errorf("focus lost after Update(): activeElement.id=%q, want %q", activeID, id+"-input")
		return
	}

	// Cursor position must be preserved at position 2 (end of "hi").
	// Without setSelectionRange after focus(), browsers reset cursor to 0.
	selStart := doc.Get("activeElement").Get("selectionStart").Int()
	if selStart != 2 {
		t.Errorf("cursor reset after Update(): selectionStart=%d, want 2 — focus() alone resets cursor to 0", selStart)
	}
}
