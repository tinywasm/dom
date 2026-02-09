//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
	. "github.com/tinywasm/dom/html"
)

type CounterComp struct {
	dom.BaseComponent
	count int
}

func (c *CounterComp) Render() dom.Node {
	return Div(
		ID(c.ID()),
		Span(ID(c.ID()+"-val"), Text("Count")),
		Button(ID(c.ID()+"-btn"), OnClick(func(e dom.Event) {
			c.count++
			dom.Update(c)
		})),
	)

}

func (c *CounterComp) RenderHTML() string {
	return ""
}

func TestBuilderAndUpdate(t *testing.T) {
	_ = dom.SetupDOM(t)

	t.Run("Render using Builder", func(t *testing.T) {
		c := &CounterComp{}
		c.SetID("counter")
		err := dom.Render("root", c)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		_, ok := dom.Get("counter-val")
		if !ok {
			t.Fatal("Counter value element not found")
		}
	})

	t.Run("Update Component", func(t *testing.T) {
		c := &CounterComp{count: 0}
		c.SetID("counter2")
		dom.Render("root", c)

		c.count = 5
		err := dom.Update(c)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		_, ok := dom.Get("counter2-val")
		if !ok {
			t.Error("Counter value element lost after update")
		}
	})

	t.Run("Append Component", func(t *testing.T) {
		// Create a parent container
		root, _ := dom.Get("root")
		root.AppendHTML(`<div id="list-container"></div>`)

		c := &CounterComp{count: 10}
		c.SetID("counter-append")

		err := dom.Append("list-container", c)
		if err != nil {
			t.Fatalf("Append failed: %v", err)
		}

		// Verify it exists in DOM
		el, ok := dom.Get("counter-append-val")
		if !ok {
			t.Fatal("Appended component element not found")
		}

		// Verify content (mock DOM doesn't strictly track structure but AppendHTML calls insertAdjacentHTML)
		// We can at least check if the element is in cache/DOM.
		_ = el
	})
}
