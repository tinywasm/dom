//go:build wasm

package main

import (
	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

// Example 1: Simple counter with Elm architecture
type Counter struct {
	dom.BaseComponent
	count int
}

func (c *Counter) Render() dom.Node {
	return dom.Div().
		Class("counter").
		Add(
			dom.Button().
				Text("-").
				OnClick(c.Decrement),
			dom.Span().
				Class("count").
				Text(fmt.Sprint(c.count)),
			dom.Button().
				Text("+").
				OnClick(c.Increment),
		).
		ToNode()
}

func (c *Counter) Increment(e dom.Event) {
	c.count++
	c.Update()
}

func (c *Counter) Decrement(e dom.Event) {
	c.count--
	c.Update()
}

func (c *Counter) OnMount() {
	fmt.Println("Counter mounted with ID:", c.GetID())
}

// Example 2: Static component with string HTML
type Header struct {
	dom.BaseComponent
}

func (h *Header) RenderHTML() string {
	return `<header class="app-header">
        <h1>DOM Refactor Example</h1>
    </header>`
}

func main() {
	// Render static header into body
	header := &Header{}
	dom.Render("body", header)

	// Append dynamic counter to body
	counter := &Counter{count: 0}
	dom.Append("body", counter)

	fmt.Println("App mounted successfully")
	select {}
}
