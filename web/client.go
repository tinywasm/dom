//go:build wasm

package main

import (
	"github.com/tinywasm/dom"
	. "github.com/tinywasm/dom/html" // DSL for UI
	"github.com/tinywasm/fmt"
)

type Counter struct {
	dom.BaseComponent // Optional: Provides ID() and SetID()
	count             int
}

func NewCounter() *Counter {
	return &Counter{}
}

// ID() is provided by dom.BaseComponent

func (c *Counter) Render() dom.Node {
	return Div(
		ID(c.ID()),
		Span(ID(c.ID()+"-val"), Text(fmt.Sprint(c.count))),
		Button(Text("Increment"), OnClick(func(e dom.Event) {
			c.count++
			dom.Update(c)
		})),
	)
}

func main() {
	dom.Render("body", NewCounter()) // Auto-generates ID if empty
	select {}
}
