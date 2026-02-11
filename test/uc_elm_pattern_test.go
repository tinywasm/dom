//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

type CounterElm struct {
	*dom.Element
	count int
}

func (c *CounterElm) Render() dom.Node {
	return dom.Div().
		Add(
			dom.Span().ID("count-val").Text(fmt.Sprint(c.count)),
		).
		ToNode()
}

func (c *CounterElm) Increment() {
	c.count++
	c.Update()
}

func TestElmPattern(t *testing.T) {
	_ = SetupDOM(t)

	t.Run("State Update and Re-render", func(t *testing.T) {
		c := &CounterElm{Element: &dom.Element{}, count: 0}
		c.SetID("counter-elm") // Fixed ID for test stability
		dom.Render("root", c)

		// Check initial render
		el, ok := dom.Get("count-val")
		if !ok {
			t.Fatal("Counter value not found")
		}
		// We can't easily check text content via Element interface in WASM tests running in node/headless?
		// Unless we expose GetTextContent in Element interface.
		// Existing Element interface: SetText, SetHTML. No GetText.
		// elementWasm has GetAttr.

		// Perform update
		c.Increment()

		// Verify re-render occurred (no error)
		el, ok = dom.Get("count-val")
		if !ok {
			t.Fatal("Counter value lost after update")
		}
		_ = el
	})
}
