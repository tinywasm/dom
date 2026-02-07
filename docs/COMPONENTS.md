# Creating Components

TinyDOM components are simple Go structs. They don't require a complex build step or special syntax. They just return strings and handle events.

## Basic Component

A basic component needs an ID and any state it needs to display. You can use the global `dom` functions in `OnMount()` to interact with elements.

```go
type Counter struct {
    dom.BaseComponent
    count int
}

func NewCounter() *Counter {
    return &Counter{}
}

// ID() and SetID() are inherited from dom.BaseComponent

func (c *Counter) RenderHTML() string {
    // Note: We manually inject the ID into the root element.
    return `
        <div id="` + c.ID() + `" class="counter">
            <span id="` + c.ID() + `-val">` + fmt.Sprint(c.count) + `</span>
            <button id="` + c.ID() + `-btn">Increment</button>
        </div>
    `
}

func (c *Counter) OnMount() {
    // 1. Get references to elements we need to interact with using global API
    valEl, _ := dom.Get(c.ID() + "-val")
    btnEl, _ := dom.Get(c.ID() + "-btn")

    // 2. Bind events
    btnEl.Click(func(e dom.Event) {
        c.count++
        // 3. Direct Update: Update only the text node, preserving the rest of the DOM
        valEl.SetText(c.count)
    })
}

func (c *Counter) OnUnmount() {}
```

## Nested Components (Manual Cascade)

TinyDOM does not automatically mount child components. You must explicitly include their HTML and call their `OnMount` method. This gives you full control over the initialization order.

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

func (p *Page) RenderHTML() string {
    return `
        <div id="` + p.ID() + `" class="page">
            <h1>My Page</h1>
            <!-- Include Child HTML -->
            ` + p.counter.RenderHTML() + `
        </div>
    `
}

func (p *Page) OnMount() {
    // Initialize Child
    p.counter.OnMount()
    
    // Page-specific logic...
}

func (p *Page) OnUnmount() {
    p.counter.OnUnmount()
}
```

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

In your `RenderHTML`, you can then use the icon:
```go
func (c *MyComponent) RenderHTML() string {
    return `<svg class="icon"><use href="#my-icon-id"></use></svg>`
}
```

## Build Tags & Separation (SSR vs WASM)

To keep WASM binaries tiny, separate your component logic using build tags:

1.  **Main File** (`comp.go`): Interface, struct, and `RenderHTML`.
2.  **SSR File** (`ssr.go`): `//go:build !wasm`. Define `RenderCSS` and `IconSvg` here.
3.  **WASM File** (`front.go`): `//go:build wasm`. Define `OnMount` and event logic here.

