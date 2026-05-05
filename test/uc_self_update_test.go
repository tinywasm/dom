//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

// ---------- Scenario 1: Self-update only ----------

// SelfUpdater simulates a component that wires listeners in OnMount
// and calls c.Update() from within one of those listeners.
type SelfUpdater struct {
	dom.Element
	selected    string
	filterTerm  string
	InputFired  int
	SelectFired int
}

func (c *SelfUpdater) Render() *dom.Element {
	return dom.Div(
		dom.Input("search").
			ID(c.GetID()+"-search").
			Attr("value", c.filterTerm).
			On("input", func(e dom.Event) {
				c.filterTerm = e.TargetValue()
				c.InputFired++
				c.Update()
			}),
		dom.Div().
			ID(c.GetID()+"-options").
			On("click", func(e dom.Event) {
				c.selected = e.TargetID()
				c.SelectFired++
				c.Update()
			}).
			Add(dom.Div().ID(c.GetID()+"-opt-a").Attr("data-id", "a").Text("Option A")),
		dom.P().ID(c.GetID()+"-result").Text(c.selected),
	)
}

// TestSelfUpdateRewiresOnMountListeners — single-component self-update path.
func TestSelfUpdateRewiresOnMountListeners(t *testing.T) {
	SetupDOM(t)

	c := &SelfUpdater{}
	if err := dom.Render("root", c); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	id := c.GetID()

	TriggerEvent(id+"-opt-a", "click", "")
	if c.SelectFired != 1 {
		t.Fatalf("first click: expected SelectFired=1, got %d", c.SelectFired)
	}

	TriggerEvent(id+"-search", "input", "hello")
	if c.InputFired != 1 {
		t.Errorf("after first Update: expected InputFired=1, got %d — search listener lost", c.InputFired)
	}

	TriggerEvent(id+"-opt-a", "click", "")
	if c.SelectFired != 2 {
		t.Errorf("after search Update: expected SelectFired=2, got %d — click listener lost", c.SelectFired)
	}
}

// ---------- Scenario 2: Parent-update THEN self-update (mirrors SelectSearch) ----------

// SSParent owns SSChild as a struct field — SelectSearch usage pattern.
type SSParent struct {
	dom.Element
	child    SSChild
	Selected string
}

func (p *SSParent) Render() *dom.Element {
	return dom.Div(
		&p.child,
		dom.P("Selected: ", p.Selected),
	)
}

// SSChild emulates SelectSearch:
//   - OnMount wires listener on `<id>-options` (event delegation).
//   - On click, it invokes a callback that updates the PARENT,
//     then calls c.Update() on itself.
type SSChild struct {
	dom.Element
	OnSelect    func(id string)
	SelectFired int
	InputFired  int
}

func (c *SSChild) Render() *dom.Element {
	return dom.Div(
		dom.Input("search").
			ID(c.GetID()+"-search").
			On("input", func(e dom.Event) {
				c.InputFired++
				c.Update()
			}),
		dom.Div().
			ID(c.GetID()+"-options").
			On("click", func(e dom.Event) {
				c.SelectFired++
				if c.OnSelect != nil {
					c.OnSelect(e.TargetID()) // triggers parent.Update()
				}
				c.Update() // then self-update
			}).
			Add(
				dom.Div().ID(c.GetID()+"-opt-a").Attr("data-id", "a").Text("Option A"),
				dom.Div().ID(c.GetID()+"-opt-b").Attr("data-id", "b").Text("Option B"),
			),
	)
}

// TestParentThenSelfUpdate reproduces the exact SelectSearch flow:
// click handler invokes parent.Update() (unmount + remount of child)
// AND THEN calls child.Update() (self-update). After this combined
// flow, the listeners must still respond on subsequent interactions.
func TestParentThenSelfUpdate(t *testing.T) {
	SetupDOM(t)

	p := &SSParent{}
	p.child.OnSelect = func(id string) {
		p.Selected = id
		p.Update()
	}
	if err := dom.Render("root", p); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	childID := p.child.GetID()

	// First click: triggers parent update, then child self-update.
	TriggerEvent(childID+"-opt-a", "click", "")
	if p.child.SelectFired != 1 {
		t.Fatalf("first click: expected SelectFired=1, got %d", p.child.SelectFired)
	}
	if p.Selected != childID+"-opt-a" {
		t.Errorf("first click: expected Selected=%q, got %q", childID+"-opt-a", p.Selected)
	}

	// Second click: must still work (this is where the bug shows).
	TriggerEvent(childID+"-opt-b", "click", "")
	if p.child.SelectFired != 2 {
		t.Errorf("second click: expected SelectFired=2, got %d — click listener lost after parent+self update",
			p.child.SelectFired)
	}

	// Search input must also still work.
	TriggerEvent(childID+"-search", "input", "abc")
	if p.child.InputFired != 1 {
		t.Errorf("search after clicks: expected InputFired=1, got %d — input listener lost", p.child.InputFired)
	}

	// Third click: confirm listeners survive multiple combined updates.
	TriggerEvent(childID+"-opt-a", "click", "")
	if p.child.SelectFired != 3 {
		t.Errorf("third click: expected SelectFired=3, got %d", p.child.SelectFired)
	}
}
