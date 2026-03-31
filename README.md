# tinywasm/dom
<img src="docs/img/badges.svg">

> **Ultra-minimal DOM & event toolkit for Go (TinyGo WASM-optimized).**

tinywasm/dom provides a minimalist, WASM-optimized way to interact with the browser DOM in Go, avoiding the overhead of the standard library and `syscall/js` exposure. It is designed specifically for **TinyGo** applications where binary size and performance are critical.

## 🚀 Features

*   **JSX-like Declarative View**: Concise nesting with `Div(H1("Title"), P("..."))`
*   **Void Element Fix**: Correctly renders `<br>`, `<img>`, `<hr>` without closing tags
*   **TinyGo Optimized**: Avoids heavy standard library packages to keep WASM binaries <500KB
*   **Direct DOM Manipulation**: No Virtual DOM overhead. You control the updates.
*   **ID-Based Caching**: Efficient element lookup and caching strategy
*   **Lifecycle Hooks**: `OnMount`, `OnUpdate`, `OnUnmount` for fine-grained control

## 📦 Installation

```bash
go get github.com/tinywasm/dom
```

## ⚡ Quick Start

For a complete example including Elm architecture (Dynamic Components) and Static Components, check the following file:

👉 **[web/client.go](web/client.go)**

This file contains the reference implementation used for testing and demonstrations.

## 🎨 JSX-like Builder API

The API allows concise nesting and typed chaining:

```go
import . "github.com/tinywasm/dom"

Div(
	H1("Welcome"),
	P("Select an option below:"),
	Ul(
		Li(Button("Action 1").On("click", handleAction1)),
		Li(Button("Action 2").On("click", handleAction2)),
	),
).Class("container")
```

**Available builders**:
- **Containers**: `Div`, `Span`, `P`, `H1`-`H6`, `Ul`, `Ol`, `Li`, `Section`, `Main`, `Article`, `Header`, `Footer`, `Nav`, `Aside`, `Table`, `Thead`, `Tbody`, `Tr`, `Td`, etc.
- **Specialized**: `Button`, `A`, `Option`, `SelectedOption`, `Fieldset`, `Legend`, `Label`.
- **SVG**: `Svg`, `Use`.

> **Note**: Form elements with validation live in `github.com/tinywasm/form`. `dom` provides basic layout and generic elements.
- **Void Elements**: `Img`, `Br`, `Hr`.

## 🔄 Lifecycle Hooks

Components can implement optional lifecycle interfaces:

```go
type MyComponent struct {
	*dom.Element
	data []string
}

// Called after component is mounted to DOM
func (c *MyComponent) OnMount() {
	c.data = fetchData()
	c.Update()
}

// Called after re-render (dom.Update)
func (c *MyComponent) OnUpdate() {
	fmt.Println("Component updated")
}

// Called before component is removed
func (c *MyComponent) OnUnmount() {
	// Cleanup resources
}
```

## 📝 Component Interface

All components must implement:

```go
type Component interface {
	GetID() string
	SetID(string)
	RenderHTML() string  // OR Render() *Element
	Children() []Component
}
```

**Two rendering options**:
1. **`RenderHTML() string`** - For static components (smaller binary)
2. **`Render() *dom.Element`** - For dynamic components (type-safe, composable)

Components can implement **either or both**. DOM checks `Render()` first, falls back to `RenderHTML()`.

## 🎯 Hybrid Rendering Strategy

Choose the right rendering method for each component:

| Component Type | Method | Benefit |
|---------------|--------|---------|
| **Static** (no interactivity) | `RenderHTML() string` | Smaller binary, less overhead |
| **Dynamic** (interactive, state) | `Render() *dom.Element` | Type-safe, composable, fluent API |

See the implementation examples in **[web/client.go](web/client.go)** to see both approaches in action.

## 🧩 Nested Components

Components can contain child components:

```go
type MyList struct {
	*dom.Element
	items []dom.Component
}

func (c *MyList) Children() []dom.Component {
	return c.items
}

func (c *MyList) Render() *dom.Element {
	list := dom.Div()
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

Embedding `*dom.Element` provides these methods automatically:

```go
type Counter struct {
	*dom.Element
	count int
}

// Chainable helpers
counter.Update()              // Trigger re-render
counter.GetID()               // Get unique ID
counter.SetID("my-id")        // Set custom ID
```

## 📚 Documentation

For more detailed information, please refer to the documentation in the `docs/` directory:

1.  **[Architecture & Builder API Guide](docs/ARCHITECTURE.md)**: Comprehensive LLM-optimized guide covering the isomorphic component model, the JSX-like builder, event handling, and optimization strategies for TinyGo.
## 🆕 What's New in v0.5.0

- ✅ **Major API Redesign** - JSX-like factories (`Div(H1("Title"))`)
- ✅ **Internal Privatization** - Cleaned up public API (privatized `EventHandler`, etc.)
- ✅ **Void Element Rendering** - Correct HTML for `<br>`, `<img>`, `<hr>`
- ✅ **Auto-ID Generation** - Simplified IDs without `auto-` prefix

- ✅ **JSX-like factories** - Concise nesting (`Div(H1("Title"), P("..."))`)
- ✅ **Void Element Rendering** - Correct HTML for `<br>`, `<img>`, `<hr>`
- ✅ **Fluent Builder API** - Chainable methods (`dom.Div().ID("x").Class("y")`)
- ✅ **Hybrid rendering** - Choose DSL or string HTML per component
- ✅ **Lifecycle hooks** - `OnMount`, `OnUpdate`, `OnUnmount`
- ✅ **Auto-ID generation** - All components get unique IDs automatically

## 📊 Performance

**Binary Size** (TinyGo WASM):
- Simple counter app: ~35KB (compressed)
- Todo list with 10 components: ~120KB (compressed)
- Full application: <500KB (compressed)

**Compared to standard library approach**: 60-80% smaller binaries.

## License

MIT
