//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

// SearchChild simulates a component with OnMount input listener (like selectsearch).
type SearchChild struct {
	dom.Element
	FilterTerm  string
	InputEvents int // counts how many times the input handler fired
}

func (c *SearchChild) Render() *dom.Element {
	return dom.Div(
		dom.Input("search").ID(c.GetID()+"-search").Attr("value", c.FilterTerm),
	)
}

func (c *SearchChild) OnMount() {
	if el, ok := dom.Get(c.GetID() + "-search"); ok {
		el.On("input", func(e dom.Event) {
			c.FilterTerm = e.TargetValue()
			c.InputEvents++
			c.Update()
		})
	}
}

// ParentWithChild holds a SearchChild in state (not created inside Render).
type ParentWithChild struct {
	dom.Element
	child    SearchChild
	updates  int
}

func (p *ParentWithChild) Render() *dom.Element {
	return dom.Div(
		&p.child,
		dom.P("updates: ", p.updates),
	)
}

func TestChildListenersAfterParentUpdate(t *testing.T) {
	SetupDOM(t)

	parent := &ParentWithChild{}
	if err := dom.Render("root", parent); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	childID := parent.child.GetID()
	searchID := childID + "-search"

	// Verify OnMount wired the listener — first input event before any Update.
	TriggerEvent(searchID, "input", "ab")
	if parent.child.InputEvents != 1 {
		t.Errorf("before Update: expected 1 input event, got %d", parent.child.InputEvents)
	}

	// Simulate parent updating (e.g. from some parent-level state change).
	parent.updates++
	parent.Update()

	// BUG: after parent Update(), child input listener should still work.
	// The child element is re-rendered with the same ID so the DOM element exists,
	// but the input handler may have been lost.
	TriggerEvent(searchID, "input", "abc")
	if parent.child.InputEvents != 2 {
		t.Errorf("after parent Update: expected 2 input events, got %d — listener lost after parent re-render", parent.child.InputEvents)
	}
}
