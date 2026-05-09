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
import . "github.com/tinywasm/dom"

Div(
    H1("Welcome"),
    P("This is a minimalist UI."),
    Div(
        Strong("Ready to start?"),
    ).Class("header-box"),
    Button("Get Started").Class("primary").On("click", func(e Event) {
        Log("Button clicked!")
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

### Component Patterns: Declarative Wiring (The Canonical Way)

The canonical way to build components is to describe the entire UI and its behavior inside `Render()`.

1.  **Events in Render**: Attach event listeners directly to elements using `.On(eventType, handler)`. The framework handles re-wiring automatically during `Update()`.
2.  **Type-safe Pairing**: Use `.For(other *Element)` for `<label for>` pairing instead of hardcoded strings. It auto-generates IDs lazily.
3.  **Closures for Lists**: Use Go closures to capture state (like loop variables) for dynamic lists.

```go
func (c *MyComponent) Render() *dom.Element {
    toggle := dom.Input("checkbox").Class("toggle")

    // Declarative wiring:
    header := dom.Label().
        For(toggle).                      // Type-safe pairing
        Text("Click me").
        On("click", c.onHeaderClick)       // Method reference

    list := dom.Div()
    for _, item := range c.Items {
        item := item // Capture for closure
        list.Add(dom.Div(item.Name).On("click", func(e dom.Event) {
            c.SelectItem(item) // Closure capture
        }))
    }

    return dom.Div(toggle, header, list)
}
```

Avoid using `OnMount()` for internal event wiring. `OnMount()` should be reserved for third-party JS integration or measuring DOM geometry.
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

## 7. Theme API

`tinywasm/dom` provides a bridge to manage the application's visual theme via the `data-theme` attribute on the `<html>` element.

```go
type Theme string

const (
    ThemeAuto  Theme = "auto"  // Removes override, follows OS preference
    ThemeDark  Theme = "dark"  // Sets data-theme="dark"
    ThemeLight Theme = "light" // Sets data-theme="light"
)

func SetTheme(theme Theme)
func GetTheme() Theme
```

The canonical `theme.css` uses these values to apply color tokens.

## 8. LocalStorage API

Since only `tinywasm/dom` is allowed to import `syscall/js`, it provides a type-safe wrapper for the browser's `localStorage`.

```go
func LocalStorageGet(key string) string
func LocalStorageSet(key, value string)
func LocalStorageDel(key string)
func LocalStorageClear()
```

**Note**: `LocalStorageGet` returns an empty string `""` if the key does not exist or if storage is unavailable.

## 9. Default Theme (`RootCSS`)

`dom/ssr.go` ships the default `:root { … }` theme of the framework via a single static function:

```go
//go:build !wasm

package dom

import _ "embed"

//go:embed theme.css
var rootCSS string

func RootCSS() string { return rootCSS }
```

`theme.css` is the **single source of truth** for the default tokens — colors, spacing, layout heights, dark-mode media query. There is no `CssVars` struct, no `DefaultCssVars()` constructor, no programmatic builder; the theme is plain CSS.

### Override

`dom` does not import `assetmin`. The contract is the function name `RootCSS`. `tinywasm/assetmin` discovers it via AST extraction during `LoadSSRModules()` and routes the result to the `open` slot of `<head>`.

Apps override the default by exposing their own `RootCSS()` from the project root's `ssr.go`. The single-override rule lives in `assetmin` (root project wins, dom is fallback, third-party modules are ignored with a warning). See [`assetmin/docs/SSR.md`](../../assetmin/docs/SSR.md).

### Distinction from `CSSProvider`

- `RootCSS()` (free function in `ssr.go`) → document-level `:root` tokens, single winner.
- `CSSProvider.RenderCSS()` (component method) → per-component scoped styles, accumulate normally.

These are intentionally separate: theme tokens are global and must not stack, while component styles are local and naturally compose.
