# Creating Components

TinyDOM components are simple Go structs. They don't require a complex build step or special syntax. They just return strings and handle events.

## Basic Component

A basic component needs an ID and any state it needs to display. You can use the global `dom` functions in `OnMount()` to interact with elements.

```go
type Counter struct {
    id    string
    count int
}

func NewCounter(id string) *Counter {
    return &Counter{id: id}
}

func (c *Counter) ID() string { return c.id }

func (c *Counter) RenderHTML() string {
    // Note: We manually inject the ID into the root element.
    return `
        <div id="` + c.id + `" class="counter">
            <span id="` + c.id + `-val">` + Convert(c.count).String() + `</span>
            <button id="` + c.id + `-btn">Increment</button>
        </div>
    `
}

func (c *Counter) OnMount() {
    // 1. Get references to elements we need to interact with using global API
    valEl, _ := dom.Get(c.id + "-val")
    btnEl, _ := dom.Get(c.id + "-btn")

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
    id      string
    counter *Counter // Child component
}

func NewPage(id string) *Page {
    return &Page{
        id:      id,
        counter: NewCounter(id + "-counter"), // Create child with sub-ID
    }
}

func (p *Page) ID() string { return p.id }

func (p *Page) RenderHTML() string {
    return `
        <div id="` + p.id + `" class="page">
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
//go:build !wasm


func (c *Counter) RenderCSS() string {
    return `
        .counter { padding: 10px; border: 1px solid #ccc; }
        .counter button { cursor: pointer; }
    `
}
```
