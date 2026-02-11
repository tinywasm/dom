//go:build wasm

package main

import (
	. "github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

// Example 1: Simple counter with Elm architecture
type Counter struct {
	*Element
	count int
}

func (c *Counter) Render() *Element {
	return Div().
		Class("counter").
		Add(
			Button().
				Text("-").
				OnClick(c.Decrement),
			Span().
				Class("count").
				Text(fmt.Sprint(c.count)),
			Button().
				Text("+").
				OnClick(c.Increment),
		)
}

func (c *Counter) Increment(e Event) {
	c.count++
	c.Update()
}

func (c *Counter) Decrement(e Event) {
	c.count--
	c.Update()
}

func (c *Counter) OnMount() {
	fmt.Println("Counter mounted with ID:", c.GetID())
}

// Example 2: Static component with string HTML
type Header struct {
	*Element
}

func (h *Header) RenderHTML() string {
	return `<header class="app-header">
        <h1>DOM Refactor Example</h1>
    </header>`
}

func main() {
	// Render static header into body
	header := &Header{Element: &Element{}}
	Render("body", header)

	// Append dynamic counter to body
	counter := &Counter{Element: &Element{}, count: 0}
	Append("body", counter)

	fmt.Println("App mounted successfully")
	select {}
}
