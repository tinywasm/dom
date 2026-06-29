# tinywasm/dom
<img src="docs/img/badges.svg">

> **Ultra-minimal DOM & event toolkit for Go (TinyGo WASM-optimized).**

tinywasm/dom provides a minimalist, WASM-optimized way to interact with the browser DOM in Go, avoiding the overhead of the standard library and `syscall/js` exposure. It is designed specifically for **TinyGo** applications where binary size and performance are critical.

## 🚀 Features

*   **Lifecycle & DOM API**: `Render`, `Append`, `Update`, `Get`, `OnHashChange`
*   **Void Element Fix**: Correctly renders `<br>`, `<img>`, `<hr>` without closing tags
*   **TinyGo Optimized**: Avoids heavy standard library packages to keep WASM binaries <500KB
*   **Fine-Grained Reactivity**: Surgical DOM patches via Signals (`SignalString`, `SignalBool`).
*   **No Virtual DOM**: Zero diffing overhead; O(1) updates that preserve focus and IME.
*   **Auto-tracking**: No manual dependency lists; signals discover their own observers.
*   **ID-Based Caching**: Efficient element lookup and caching strategy.

## 📦 Installation

```bash
go get github.com/tinywasm/dom
```

## ⚡ Quick Start

For a complete example including Elm architecture (Dynamic Components) and Static Components, see:

👉 **[tinywasm/html — web/client.go](https://github.com/tinywasm/html/blob/main/web/client.go)**

That file uses `tinywasm/html` for element builders and `tinywasm/dom` for lifecycle (Render, Update, Append).

## 📦 Related Packages

`tinywasm/dom` focuses on DOM manipulation and component lifecycles. HTML element builders have been moved to their own packages:

- [tinywasm/html](https://github.com/tinywasm/html) — HTML element builders (Div, Span, Nav...)
- [tinywasm/svg](https://github.com/tinywasm/svg) — SVG builders + icon sprite system
- [tinywasm/image](https://github.com/tinywasm/image) — Image element builders

## 🔄 Component Lifecycle

Components can optionally implement the `Init(dom.Ctx)` hook:

```go
type MyComponent struct {
	dom.Element
	name *dom.SignalString
}

// Called once when component is first mounted
func (c *MyComponent) Init(ctx dom.Ctx) {
	c.name = dom.NewString("World")
	ctx.OnCleanup(func() {
		// Cleanup resources (timers, etc)
	})
}

func (c *MyComponent) Render() *dom.Element {
	return html.Div(
		html.Span("Hello, ").BindText(c.name),
	)
}
```

## 📝 Component Interface

All components must implement:

```go
type Component interface {
	GetID() string
	SetID(string)
	String() string  // OR Render() *Element
	Children() []Component
}
```

**Two rendering options**:
1. **`String() string`** - For static components (smaller binary)
2. **`Render() *dom.Element`** - For dynamic components (type-safe, composable)

Components can implement **either or both**. DOM checks `Render()` first, falls back to `String()`.

## 🎯 Hybrid Rendering Strategy

Choose the right rendering method for each component:

| Component Type | Method | Benefit |
|---------------|--------|---------|
| **Static** (no interactivity) | `String() string` | Smaller binary, less overhead |
| **Dynamic** (interactive, state) | `Render() *dom.Element` | Type-safe, composable, fluent API |

See the implementation examples in **[web/client.go](web/client.go)** to see both approaches in action.

## 🧩 Nested Components

Components can contain child components:

```go
type MyList struct {
	dom.Element
	items []dom.Component
}

func (c *MyList) Children() []dom.Component {
	return c.items
}

func (c *MyList) Render() *dom.Element {
	list := html.Div()
	for _, item := range c.items {
		list.Add(item) // Components can be children
	}
	return list
}
```

When you call `dom.Render("app", myList)`, the library will:
1. Render the HTML
2. Call `OnMount()` for `MyList`
3. Recursively call `OnMount()` for all `items`

The same recursion applies to cleanup, ensuring all event listeners are cleaned up when a parent is replaced.

## 🎯 Event Handling

Event handling is integrated directly into the Builder API via `On(eventType, handler)`.


## 🔧 Core API

### Package Functions

```go
// Rendering
dom.Render(parentID, component)  // Replace parent's content
dom.Append(parentID, component)  // Append after last child
dom.Update(component)            // Re-render in place
dom.Get(id)                      // Get a DOM Reference (value, focus, events)

// Routing (hash-based)
dom.OnHashChange(handler)        // Listen to hash changes
dom.GetHash()                    // Get current hash
dom.SetHash(hash)                // Set hash
```

### Element Helpers

Embedding `dom.Element` provides these methods automatically:

```go
type Counter struct {
	dom.Element
	count int
}

// Chainable helpers
counter.Update()              // Trigger re-render
counter.GetID()               // Get unique ID
counter.SetID("my-id")        // Set custom ID
```

## 📚 Documentation

For more detailed information, please refer to the documentation in the `docs/` directory:

1.  **[Architecture & Builder API Guide](docs/ARCHITECTURE.md)**: Comprehensive guide covering the isomorphic component model, the JSX-like builder, event handling, and optimization strategies for TinyGo.
2.  **[Design Decisions](docs/DESIGN.md)**: Rationale for Signals, no generics, and the construction harness.
3.  **[Lifecycle Diagram](docs/diagrams/lifecycle.md)**: Mermaid flowchart of the component lifecycle.
4.  **[Binding Model](docs/BINDING_MODEL.md)**: Mental model of how reactive state updates the DOM.
5.  **[Trade-offs](docs/TRADEOFFS.md)**: Pros/cons of fine-grained reactivity vs VDOM.
6.  **[Agent Guide](AGENTS.md)**: Constraints and rules for agents and human contributors.
## 🆕 What's New in v0.5.0

- ✅ **Major API Redesign** - Builders moved to separate packages
- ✅ **Interface Standardized** - `RenderHTML() string` → `String() string` (`fmt.Stringer`)
- ✅ **Internal Privatization** - Cleaned up public API (privatized `EventHandler`, etc.)
- ✅ **Void Element Rendering** - Correct HTML for `<br>`, `<img>`, `<hr>`
- ✅ **Auto-ID Generation** - Simplified IDs without `auto-` prefix

- ✅ **Void Element Rendering** - Correct HTML for `<br>`, `<img>`, `<hr>`
- ✅ **Fluent Builder API** - Chainable methods (`html.Div().ID("x").Class("y")`)
- ✅ **Hybrid rendering** - Choose DSL or string HTML per component
- ✅ **Fine-Grained Reactivity** - Typed signals (`SignalString`, `SignalBool`, `SignalNodes`)
- ✅ **Auto-ID generation** - All components get unique IDs automatically

## 📊 Performance

**Binary Size** (TinyGo WASM):
- Simple counter app: ~35KB (compressed)
- Todo list with 10 components: ~120KB (compressed)
- Full application: <500KB (compressed)

**Compared to standard library approach**: 60-80% smaller binaries.

## License

MIT
