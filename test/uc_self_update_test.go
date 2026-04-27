//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

// SelfUpdater simulates a component that wires listeners in OnMount
// and calls c.Update() from within one of those listeners (e.g. SelectSearch).
type SelfUpdater struct {
	dom.Element
	selected    string
	filterTerm  string
	InputFired  int
	SelectFired int
}

func (c *SelfUpdater) Render() *dom.Element {
	return dom.Div(
		dom.Input("search").ID(c.GetID()+"-search").Attr("value", c.filterTerm),
		dom.Div().ID(c.GetID()+"-options").
			Add(dom.Div().ID(c.GetID()+"-opt-a").Attr("data-id", "a").Text("Option A")),
		dom.P().ID(c.GetID()+"-result").Text(c.selected),
	)
}

func (c *SelfUpdater) OnMount() {
	id := c.GetID()

	// Search listener
	if searchEl, ok := dom.Get(id + "-search"); ok {
		searchEl.On("input", func(e dom.Event) {
			c.filterTerm = e.TargetValue()
			c.InputFired++
			c.Update()
		})
	}

	// Options click listener
	if optEl, ok := dom.Get(id + "-options"); ok {
		optEl.On("click", func(e dom.Event) {
			c.selected = e.TargetID()
			c.SelectFired++
			c.Update()
		})
	}
}

// TestSelfUpdateRewiresOnMountListeners verifies that after a component calls
// c.Update() from within an OnMount listener, subsequent interactions still
// trigger their respective listeners (both search input and option click).
//
// Bug: Update() calls cleanupListeners(id) but never re-calls OnMount() on the
// component itself, so all dynamically-wired listeners are permanently lost.
func TestSelfUpdateRewiresOnMountListeners(t *testing.T) {
	SetupDOM(t)

	c := &SelfUpdater{}
	if err := dom.Render("root", c); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	id := c.GetID()

	// --- First interaction: click an option ---
	TriggerEvent(id+"-opt-a", "click", "")
	if c.SelectFired != 1 {
		t.Fatalf("first option click: expected SelectFired=1, got %d", c.SelectFired)
	}

	// --- After Update(), search listener must still work ---
	TriggerEvent(id+"-search", "input", "hello")
	if c.InputFired != 1 {
		t.Errorf("after first Update: expected InputFired=1, got %d — search listener lost after self-update", c.InputFired)
	}

	// --- After search Update(), option click must still work ---
	TriggerEvent(id+"-opt-a", "click", "")
	if c.SelectFired != 2 {
		t.Errorf("after search Update: expected SelectFired=2, got %d — click listener lost after self-update", c.SelectFired)
	}
}
