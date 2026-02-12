# Creating Components

TinyDOM components are simple Go structs. They don't require a complex build step or special syntax. They just return strings and handle events.

## Basic Component

A basic component needs an ID and any state it needs to display. You can use the global `dom` functions in `OnMount()` to interact with elements.

```go
import (
	"fmt"

	"github.com/tinywasm/dom"
)

type Counter struct {
	*dom.Element
	count int
}

func NewCounter() *Counter {
	return &Counter{Element: dom.Div()}
}

// Render uses the declarative JSX-like API
func (c *Counter) Render() *dom.Element {
	return dom.Div(
		dom.Span(fmt.Sprint(c.count)).Class("count"),
		dom.Button("Increment").On("click", func(e dom.Event) {
			c.count++
			c.Update()
		}),
	).Class("counter")
}

// OnMount is optional if you only use inline events,
// but still available for complex logic.
func (c *Counter) OnMount() {
	dom.Log("Counter mounted:", c.GetID())
}
```

## Nested Components (Recursive Lifecycle)

TinyDOM automatically manages the lifecycle of child components. To enable this, implement the `Children()` method in your component. This allows `Mount` and `Unmount` to recursively call `OnMount` and `OnUnmount` for all descendants.

```go
type Page struct {
    *dom.Element
    counter *Counter // Child component
}

func NewPage() *Page {
    return &Page{
        Element: dom.Div(),
        counter: NewCounter(),
    }
}

```go
type Page struct {
	*dom.Element
	counter *Counter
}

// Children MUST be implemented to ensure the child's lifecycle (OnMount) is triggered.
func (p *Page) Children() []dom.Component {
	return []dom.Component{p.counter}
}

func (p *Page) Render() *dom.Element {
	return dom.Div(
		dom.H1("My Page"),
		p.counter, // Embedding a child component directly
	).Class("page")
}
```

> [!TIP]
> `dom.Element` (when embedded) provides a default implementation of `Children()` that returns `nil`, so you only need to override it if the component has children.

## CSS Handling

Since `RenderCSS` is only needed for the backend (to bundle styles), you can define it on your component struct. It will be ignored by the WASM build if you use build tags, or simply not called by the frontend logic.

```go
// ssr.go (!wasm)

func (c *Counter) RenderCSS() string {
    return `
        .counter { padding: 10px; border: 1px solid #ccc; }
        .counter button { cursor: pointer; }
    `
}
```

## SVG Icon Management (`IconSvgProvider`)

To register SVG icons in a global sprite (accessible via `<use href="#id">`), components can implement the `IconSvgProvider` interface.

> [!IMPORTANT]
> **MANDATORY:** The `IconSvg()` method MUST be in a file with the `//go:build !wasm` tag (e.g., `ssr.go`).
> SVG strings are dead code on the WASM client and unnecessarily increase the binary size.

```go
// ssr.go (!wasm)

func (c *MyComponent) IconSvg() map[string]string {
    return map[string]string{
        // Internal SVG content (paths, etc)
        // Default viewBox="0 0 16 16" unless specified.
        "my-icon-id": `<path d="..." />`, 
    }
}
```

In your `Render`, you can then use the icon helper (if you create one) or just `html.Raw`:
```go
func (c *MyComponent) Render() *dom.Element {
	return Div(
		Class("icon"),
		Raw(`<svg><use href="#my-icon-id"></use></svg>`),
	)
}
```

## Separation (SSR vs WASM)

To keep WASM binaries tiny, separate your component logic using build tags:

1.  **Main File** (`comp.go`): Interface, struct, and `Render`.
2.  **SSR File** (`ssr.go`): `//go:build !wasm`. Define `RenderCSS` and `IconSvg` here.
3.  **WASM File** (`front.go`): `//go:build wasm`. Define complex `OnMount` logic here (if needed).

