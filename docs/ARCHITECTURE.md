# `tinywasm/dom` Architecture & Builder API (LLM Context)

`tinywasm/dom` is a minimalist, dependency-free wrapper over the browser DOM, optimized for `TinyGo/WASM`. It provides a Go-native, type-safe API for building UIs without exposing `syscall/js`.

## 1. Core Principles & Philosophy
- **Isomorphic Core**: Same structs compile for server (`!wasm`) and client (`wasm`).
- **No Virtual DOM**: Uses direct `.Update()` calls instead of React-style VDOM diffing. Less memory, faster in WASM.
- **JSX-like Builder**: Strongly-typed Go functions (e.g. `dom.Div()`, `dom.Button()`) to construct the DOM tree.
- **Zero StdLib**: Uses `github.com/tinywasm/fmt` instead of `fmt`, `strings`, `errors` to reduce WASM size.
- **Slices over Maps**: Attributes and events use `[]fmt.KeyValue` instead of `map[string]string` because maps are extremely heavy in TinyGo.

## 2. API Overview

There are three primary layers/interfaces:
- **Global `dom` API**: `Render(parentID, comp)`, `Append(parentID, comp)`, `Update(comp)`.
- **`Component` Interface**: `GetID()`, `SetID(id)`, `RenderHTML()`, `Children()`.
- **`Reference` Interface**: Represents a live DOM node (Read: `GetAttr`, `Value`, `Checked`; Interaction: `On`, `Focus`).

### Mount point: always use `"app"`, never `"body"`

`Render(parentID, comp)` sets `parent.innerHTML = html`, replacing ALL existing children of the
target element. Using `"body"` as the mount point **destroys the SVG sprite** injected inline by
`tinywasm/assetmin`, breaking all `<use href="#icon-id">` references.

The `tinywasm/assetmin` HTML template already injects `<div id="app"></div>` before the `<script>`
tag. Always mount the root component there:

```go
// ✅ CORRECT — sprite SVG stays intact in <body> alongside <div id="app">
Render("app", &App{})

// ❌ WRONG — overwrites body.innerHTML, removes the SVG sprite
Render("body", &App{})
```

### The Builder (JSX-like UI construction)
Elements are constructed declaratively:
```go
import "github.com/tinywasm/dom"

dom.Div(
    dom.H1("Welcome"),
    dom.P("This is a minimalist UI."),
    dom.Div(
        dom.Strong("Ready to start?"),
    ).Class("header-box"),
    dom.Button("Get Started").Class("primary").On("click", func(e dom.Event) {
        dom.Log("Button clicked!")
    }),
).Class("container")
```
- Element factories accept `...any` children — strings, numbers, and `*Element` values. Non-string values are converted via `tinywasm/fmt.Sprint`.
- **Typing**: Some factories return specific types to allow chainable semantic methods, though most generic elements return `*Element`.

## 3. Creating Components

A component is a Go struct that embeds `dom.Element` **as a value** (never as a pointer) and implements `Render() *dom.Element`.

```go
// ✅ CORRECT — value embed: 1 allocation, no nil-panic risk, better GC in TinyGo.
type Counter struct {
	dom.Element
	count int
}

// ❌ WRONG — pointer embed: 2 allocations, nil-panic risk, heavier GC pressure.
// type Counter struct { *dom.Element; count int }

func (c *Counter) Render() *dom.Element {
	return dom.Div(
		dom.Span("Count: ", c.count).Class("count"),
		dom.Button("Increment").On("click", func(e dom.Event) {
			c.count++
			c.Update() // Triggers direct DOM replacement for this component only
		}),
	).Class("counter")
}
```

**Why value embed?** TinyGo has a simple GC — fewer heap objects means fewer pauses. Value embedding keeps the struct and its `Element` identity in a single allocation with better cache locality.

### Component Lifecycle (WASM only)
If a component implements `Mountable`, `OnMount()` is called after injection.
```go
//go:build wasm
func (c *Counter) OnMount() { dom.Log("Mounted ID:", c.GetID()) }
```
**Recursive Lifecycle**: If a component has child components, it MUST implement `Children() []dom.Component` so the framework knows to trigger the child's `OnMount`.

### Component Assets (Backend only)
To bundle styles/icons, implement these interfaces (must use `//go:build !wasm` tag):
- `CSSProvider`: `RenderCSS() string`
- `IconSvgProvider`: `IconSvg() map[string]string`

## 4. Events
The `dom.Event` interface provides safe access to the JS Event without `syscall/js`:
- `PreventDefault()`, `StopPropagation()`
- `TargetValue() string` (Extremely useful for `dom.Input("text")` and `<select>`)
- `TargetID() string`

> [!IMPORTANT]
> `dom.Input` should be used only for basic layout elements (like a toggle checkbox or a simple search box). For any input that requires validation, labels, or form state management, you MUST use `github.com/tinywasm/form/input`.

## 5. Void Elements
The library handles self-closing tags correctly for:
- `Input(type)`, `Img(src, alt)`, `Br()`, `Hr()`.
These return elements with the `void` flag set, preventing the rendering of a closing tag.

## 6. Build Split Strategy
- `dom_wasm.go` & `element_wasm.go`: Implementation using `syscall/js`.
- `dom_backend.go` & `dom_stub.go` (`!wasm`): No-op / server-side logic for compilation safety.
- **WASM Memory Safety**: `Unmount` automatically releases all saved `js.FuncOf` event listeners.
