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
		ID(c.GetID()).
		Class("counter").
		Append(
			dom.Button().
				Text("-").
				OnClick(c.Decrement),
		).
		Append(
			dom.Span().
				Class("count").
				Text(fmt.Sprint(c.count)),
		).
		Append(
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
	// Render static header
	header := &Header{}
	dom.Render("app", header)

	// Render dynamic counter
	counter := &Counter{count: 0}
	dom.Append("app", counter)

	fmt.Println("App mounted successfully")
	select {}
}
