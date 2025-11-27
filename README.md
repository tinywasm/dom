# TinyDOM

> **Ultra-minimal DOM & event toolkit for Go (TinyGo WASM-optimized).**

TinyDOM provides a minimalist, WASM-optimized way to interact with the browser DOM in Go, avoiding the overhead of the standard library and `syscall/js` exposure. It is designed specifically for **TinyGo** applications where binary size and performance are critical.

## 🚀 Features

*   **TinyGo Optimized**: Avoids heavy standard library packages like `fmt` or `net/http` to keep WASM binaries small.
*   **Direct DOM Manipulation**: No Virtual DOM overhead. You control the updates.
*   **ID-Based Caching**: Efficient element lookup and caching strategy.
*   **Memory Safe**: Automatic event listener cleanup on `Unmount`.

## 📦 Installation

```bash
go get github.com/cdvelop/tinydom
```

## ⚡ Quick Start

Here is a simple "Counter" component example.

```go
package main

import (
	"github.com/cdvelop/tinydom"
	."github.com/cdvelop/tinystring"
)

// Counter is a simple component
type Counter struct {
	id    string
	count int
}

// NewCounter creates a new instance
func NewCounter(id string) *Counter {
	return &Counter{id: id}
}

// ID returns the component's unique ID
func (c *Counter) ID() string { return c.id }

// RenderHTML returns the initial HTML structure
func (c *Counter) RenderHTML() string {
	return `
		<div id="` + c.id + `">
			<span id="` + c.id + `-val">` + Convert(c.count).String() + `</span>
			<button id="` + c.id + `-btn">Increment</button>
		</div>
	`
}

// OnMount binds event listeners
func (c *Counter) OnMount(dom tinydom.DOM) {
	valEl, _ := dom.Get(c.id + "-val")
	btnEl, _ := dom.Get(c.id + "-btn")

	btnEl.Click(func(e tinydom.Event) {
		c.count++
		valEl.SetText(Convert(c.count).String())
	})
}

// OnUnmount cleans up (optional, listeners are auto-removed)
func (c *Counter) OnUnmount() {}

func main() {
	// Initialize TinyDOM
	dom := tinydom.New(nil)
	
	// Mount the component to an existing element with id "app"
	dom.Mount("app", NewCounter("my-counter"))
	
	// Prevent main from exiting
	select {}
}
```

## 📚 Documentation

For more detailed information, please refer to the documentation in the `docs/` directory:

1.  **[Specification & Philosophy](docs/SPECIFICATION.md)**: Design goals, architecture, and key decisions.
2.  **[API Reference](docs/API.md)**: Detailed definition of `DOM`, `Element`, and `Component` interfaces.
3.  **[Creating Components](docs/COMPONENTS.md)**: Guide to building basic and nested components.
4.  **[Event Handling](docs/EVENTS.md)**: Using the `Event` interface for clicks, inputs, and forms.
5.  **[TinyString Helper](docs/TINYSTRING.md)**: Quick guide for string conversions and manipulations.
6.  **[Advanced Patterns](docs/ADVANCED.md)**: Dynamic lists, decoupling, and performance tips.
7.  **[Comparison](docs/COMPARISON.md)**: TinyDOM vs. syscall/js, VDOM, and JS frameworks.


## License

MIT