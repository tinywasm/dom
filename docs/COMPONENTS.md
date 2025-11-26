# Creating Components

TinyDOM components are simple Go structs. They don't require a complex build step or special syntax. They just return strings and handle events.

## Basic Component

A basic component needs an ID, a reference to the DOM, and any state it needs to display.

```go
type Counter struct {
    dom   tinydom.DOM
    id    string
    count int
}

func NewCounter(dom tinydom.DOM, id string) *Counter {
    return &Counter{dom: dom, id: id}
}

func (c *Counter) ID() string { return c.id }

func (c *Counter) RenderHTML() string {
    // Note: We manually inject the ID into the root element.
    return `
        <div id="` + c.id + `" class="counter">
            <span id="` + c.id + `-val">` + tinystring.Convert(c.count).String() + `</span>
            <button id="` + c.id + `-btn">Increment</button>
        </div>
    `
}

func (c *Counter) OnMount() {
    // 1. Get references to elements we need to interact with
    valEl := c.dom.Get(c.id + "-val")
    btnEl := c.dom.Get(c.id + "-btn")

    // 2. Bind events
    btnEl.Click(func() {
        c.count++
        // 3. Direct Update: Update only the text node, preserving the rest of the DOM
        valEl.SetText(tinystring.Convert(c.count).String())
    })
}
```

## Nested Components (Manual Cascade)

TinyDOM does not automatically mount child components. You must explicitly include their HTML and call their `OnMount` method. This gives you full control over the initialization order.

```go
type Page struct {
    dom     tinydom.DOM
    id      string
    counter *Counter // Child component
}

func NewPage(dom tinydom.DOM, id string) *Page {
    return &Page{
        dom:     dom,
        id:      id,
        counter: NewCounter(dom, id+"-counter"), // Create child with sub-ID
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
