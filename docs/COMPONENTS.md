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
	dom.BaseComponent
	count int
}

func NewCounter() *Counter {
	return &Counter{}
}

// Render uses the declarative Builder API
func (c *Counter) Render() dom.Node {
	return dom.Div().
		Class("counter").
		Add(
			dom.Span().
				ID(c.GetID()+"-val").
				Text(fmt.Sprint(c.count)),
			dom.Button().
				ID(c.GetID()+"-btn").
				Text("Increment").
				// Event handling is now inline and declarative!
				OnClick(func(e dom.Event) {
					c.count++
					// Update re-render triggers automatically via Update()
					c.Update()
				}),
		).
		ToNode()
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
    dom.BaseComponent
    counter *Counter // Child component
}

func NewPage() *Page {
    return &Page{
        counter: NewCounter(),
    }
}

```go
type Page struct {
	dom.BaseComponent
	counter *Counter
}

// Children MUST be implemented to ensure the child's lifecycle (OnMount) is triggered.
func (p *Page) Children() []dom.Component {
	return []dom.Component{p.counter}
}

func (p *Page) Render() dom.Node {
	return dom.Div().
		Class("page").
		Add(
			dom.H1().Text("My Page"),
			// Embedding a child component directly
			p.counter,
		).
		ToNode()
}
```

> [!TIP]
> `dom.BaseComponent` provides a default implementation of `Children()` that returns `nil`, so you only need to override it if the component has children.

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
func (c *MyComponent) Render() dom.Node {
	return Div(
		Class("icon"),
		Raw(`<svg><use href="#my-icon-id"></use></svg>`),
	)
}
```

## Build Tags & Separation (SSR vs WASM)

To keep WASM binaries tiny, separate your component logic using build tags:

1.  **Main File** (`comp.go`): Interface, struct, and `Render`.
2.  **SSR File** (`ssr.go`): `//go:build !wasm`. Define `RenderCSS` and `IconSvg` here.
3.  **WASM File** (`front.go`): `//go:build wasm`. Define complex `OnMount` logic here (if needed).

## SSR & Hydration

To avoid a flicker when your application starts, use `dom.Hydrate` for the initial server-rendered module.

1.  **Server (Backend)**: Renders the full HTML.
2.  **Client (WASM)**: Calls `dom.Hydrate` on the root element. This attaches all event listeners and triggers `OnMount` without replacing the existing DOM nodes.

```go
// main.go (WASM)
func main() {
    // Correct way to "awaken" server-rendered HTML
    dom.Hydrate("app", myRootComponent)
    select {}
}
```

