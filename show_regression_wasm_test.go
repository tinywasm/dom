//go:build wasm

package dom

import (
	"testing"
)

// TestShowSecondToggleSharedContent reproduces the panic reported in
// tinywasm/layout docs/BUG_DOM.md: every consumer of Show closes over
// *Elements built outside the render callback (modaldialog's modalContent,
// selectsearch's searchInput/optList). Show re-invokes the callback on every
// false→true transition, so the SECOND open re-attaches an element whose
// `attached` flag is already set and Child panics:
//
//	"dom: element div  is already a child of another element"
//
// The panic kills the whole Go/WASM program in production (event handlers run
// on the js event goroutine). Here the toggle runs on the test goroutine, so
// the panic is recovered and reported as a plain test failure — the failure
// IS the bug: with a harness-closed API this scenario must not be reachable.
func TestShowSecondToggleSharedContent(t *testing.T) {
	cond := NewBool(false)

	// The consumer pattern, verbatim from modaldialog.Render: the body is
	// built ONCE and captured by the Show callback, because nothing in the
	// signature func() *Element says every call must return a fresh tree.
	body := NewElement("div").ID("shared-body").
		Child(NewElement("span").ID("shared-msg").Text("Delete «laptop»?"))

	s := Show(cond, func() *Element {
		return NewElement("div").ID("modal-root").Child(body)
	})
	Render("app", s)

	// First open: works — body.attached flips false→true here.
	cond.Set(true)
	if _, ok := Get("shared-msg"); !ok {
		t.Fatal("first open: content should be in the DOM")
	}

	// Close: Show clears the DOM nodes but the Go-side tree still records
	// body as attached.
	cond.Set(false)

	// Second open: the callback re-runs Child(body) on the attached element.
	// Recover so the failure is isolated to this test instead of aborting the
	// whole WASM test binary the way it aborts the app in production.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second open panicked (the reported bug): %v", r)
		}
	}()
	cond.Set(true)
	if _, ok := Get("shared-msg"); !ok {
		t.Fatal("second open: content should be in the DOM")
	}
}
