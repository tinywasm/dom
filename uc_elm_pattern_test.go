//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

type Counter struct {
	dom.BaseComponent
	count int
}

func (c *Counter) Render() dom.Node {
	// Simple counter that renders just the number as text inside the div
	return dom.Div().ID(c.GetID()).Append(dom.Text(fmt.Sprint(c.count))).ToNode()
}

func (c *Counter) Increment() {
	c.count++
	c.Update()
}

func TestElmPattern(t *testing.T) {
	_ = dom.SetupDOM(t)

	c := &Counter{}
	c.SetID("counter")
	dom.Render("root", c)

	doc := js.Global().Get("document")
	el := doc.Call("getElementById", "counter")
	if el.IsNull() {
		t.Fatal("Counter not found")
	}
	if el.Get("innerHTML").String() != "0" {
		t.Errorf("Expected 0, got %s", el.Get("innerHTML").String())
	}

	c.Increment()

	el = doc.Call("getElementById", "counter")
	if el.IsNull() {
		t.Fatal("Counter lost after update")
	}
	if el.Get("innerHTML").String() != "1" {
		t.Errorf("Expected 1, got %s", el.Get("innerHTML").String())
	}
}
