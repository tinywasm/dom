# `tinywasm/dom` Architecture & Builder API (LLM Context)

`tinywasm/dom` is a minimalist, dependency-free wrapper over the browser DOM, optimized for `TinyGo/WASM`. It provides a Go-native, type-safe API for building UIs without exposing `syscall/js`.

## 1. Core Principles & Philosophy
- **Isomorphic Core**: Same structs compile for server (`!wasm`) and client (`wasm`).
- **No Virtual DOM**: Uses direct `.Update()` calls instead of React-style VDOM diffing. Less memory, faster in WASM.
- **JSX-like Builder**: Strongly-typed Go functions (e.g. `dom.Div()`, `dom.Input()`) to construct the DOM tree.
- **Zero StdLib**: Uses `github.com/tinywasm/fmt` instead of `fmt`, `strings`, `errors` to reduce WASM size.
- **Slices over Maps**: Attributes and events use `[]fmt.KeyValue` instead of `map[string]string` because maps are extremely heavy in TinyGo.

## 2. API Overview

There are three primary layers/interfaces:
- **Global `dom` API**: `Render(parentID, comp)`, `Append(parentID, comp)`, `Update(comp)`.
- **`Component` Interface**: `GetID()`, `SetID(id)`, `RenderHTML()`, `Children()`.
- **`Reference` Interface**: Represents a live DOM node (Read: `GetAttr`, `Value`, `Checked`; Interaction: `On`, `Focus`).

### The Builder (JSX-like UI construction)
Elements are constructed declaratively:
```go
import "github.com/tinywasm/dom"

dom.Form(
    dom.Text("username", "Enter username").Required().ID("user-id"),
    dom.Password("pwd"),
    dom.Div(
        dom.Strong("Login"),
    ).Class("header-box"),
    dom.Button("Submit").Attr("type", "submit"),
).Action("/api/login").Method("POST").OnSubmit(func(e dom.Event) {
    e.PreventDefault()
})
```
- Content methods (`SetText`, `SetHTML`) accept variadic arguments, using `fmt.Sprint` for non-strings.
- **Typing**: `dom.Input()`, `dom.Select()`, etc., return typed structs (`*InputEl`, `*SelectEl`) to allow specific builder methods (`.Required()`, `.Rows()`).

## 3. Creating Components

A component is a Go struct that embeds `*dom.Element` (for identity/lifecycle) and implements `Render() *dom.Element`.

```go
type Counter struct {
	*dom.Element
	count int
}

func NewCounter() *Counter { return &Counter{Element: dom.Div()} }

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
- `TargetValue() string` (Extremely useful for `<input>` and `<select>`)
- `TargetID() string`

## 5. Build Split Strategy
- `dom_wasm.go` & `element_wasm.go`: Implementation using `syscall/js`.
- `dom_backend.go` & `dom_stub.go` (`!wasm`): No-op / server-side logic for compilation safety.
- **WASM Memory Safety**: `Unmount` automatically releases all saved `js.FuncOf` event listeners.
