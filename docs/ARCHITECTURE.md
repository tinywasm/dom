# `tinywasm/dom` Architecture & Builder API (LLM Context)

`tinywasm/dom` is a minimalist, dependency-free wrapper over the browser DOM, optimized for `TinyGo/WASM`. It provides a Go-native, type-safe API for building UIs without exposing `syscall/js`.

## 1. Core Principles & Philosophy
- **Isomorphic Core**: Same structs compile for server (`!wasm`) and client (`wasm`).
- **No Virtual DOM**: Fine-grained reactivity via typed Signals (`SignalString`/`SignalBool`/`SignalNodes`). Signal changes patch only the bound DOM node — O(1), no diffing, no manual `Update()` calls.
- **DOM-Only Layer**: Provides the `Element` struct, lifecycle interfaces, and direct DOM manipulation. HTML element builders live in `tinywasm/html`, SVGs in `tinywasm/svg`, and images in `tinywasm/image`.
- **Zero StdLib**: Uses `github.com/tinywasm/fmt` instead of `fmt`, `strings`, `errors` to reduce WASM size.
- **Slices over Maps**: Attributes and events use `[]fmt.KeyValue` instead of `map[string]string` because maps are extremely heavy in TinyGo.

## 2. API Overview

There are three primary layers/interfaces:
- **Global `dom` API**: `Render(parentID, comp)`, `Append(parentID, comp)`, `SetDevMode(bool)`. (`update` is unexported — authors never call it; signals patch the DOM directly.)
- **`Component` Interface**: `GetID()`, `SetID(id)`, `String()`, `Children()`.
- **`Reference` Interface**: Represents a live DOM node. Read: `GetAttr`, `Value`, `Checked`. Mutation: `SetValue`, `SetAttr`, `RemoveAttr`, `SetText`. Interaction: `On`, `Focus`.

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

### Package Boundaries
| Concern | Package |
|---|---|
| HTML element builders | `tinywasm/html` |
| SVG builders + sprite | `tinywasm/svg` |
| Image builders | `tinywasm/image` |
| DOM manipulation, Element type, interfaces | `tinywasm/dom` (this package) |

Elements are constructed declaratively using builders from sibling packages:
```go
import (
    . "github.com/tinywasm/html"
    . "github.com/tinywasm/dom"
)

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

## 3. Creating Components

A component is a Go struct that embeds `dom.Element` **as a value** (never as a pointer) and implements `Render() *dom.Element`.

```go
// ✅ CORRECT — value embed: 1 allocation, no nil-panic risk, better GC in TinyGo.
type Counter struct {
	dom.Element
	count *dom.SignalString
}

func (c *Counter) Init(ctx dom.Ctx) {
	c.count = dom.NewString("0")
}

func (c *Counter) Render() *dom.Element {
	return html.Div(
		html.Span().BindText(c.count).Class("count"),
		html.Button("Increment").On("click", func(e dom.Event) {
			c.count.Update(func(v string) string {
				i, _ := strconv.Atoi(v)
				return strconv.Itoa(i + 1)
			})
		}),
	).Class("counter")
}
```

Reactivity is achieved through **Signals**. When a signal changes, only the bound DOM nodes are updated. No `Update()` calls are needed.

### Component Patterns: Declarative Wiring (The Canonical Way)

The canonical way to build components is to describe the entire UI and its behavior inside `Render()`.

1.  **Events in Render**: Attach event listeners directly to elements using `.On(eventType, handler)`.
2.  **Bindings in Render**: Use `.BindText()`, `.BindClass()`, etc., to link signals to DOM attributes or content.
3.  **Type-safe Pairing**: Use `.For(other *Element)` for `<label for>` pairing.
4.  **Autofocus**: Use `.Autofocus()` to focus an element when it first appears.

```go
func (c *MyComponent) Render() *dom.Element {
    toggle := html.Input("checkbox").Class("toggle")

    return html.Div(
        html.Label().For(toggle).Text("Click me"),
        toggle,
    )
}
```

**Why value embed?** TinyGo has a simple GC — value embedding keeps the struct and its `Element` identity in a single allocation.

### Component Lifecycle (WASM only)
1. **`Init(ctx dom.Ctx)` (Optional)**: Called once when the component is first mounted. Use it to initialize signals or register cleanups via `ctx.OnCleanup(fn)`.
2. **`Render() *dom.Element`**: Called once to build the initial DOM tree.
3. **Signal Patches**: When a signal changes, the engine surgically updates the bound DOM node.
4. **Cleanup**: When a component is unmounted, all its signal subscriptions and `OnCleanup` functions are automatically executed.

### Component Assets (Backend only)
To bundle styles/icons, implement these interfaces:
- `CSSProvider`: `RenderCSS() any` (Expected to return `*css.Stylesheet` for SSR)

## 4. Events & Bindings

### Events
The `dom.Event` interface provides safe access to the JS Event without `syscall/js`:
- `PreventDefault()`, `StopPropagation()`
- `TargetValue() string`
- `TargetID() string`

### Bindings (Reactivity)
Bindings link a `Signal` to a DOM property. When the signal's value changes, the DOM is patched automatically.

- `.BindText(s *SignalString)`: `textContent` tracks the signal.
- `.BindAttr(name string, s *SignalString)`: Attribute tracks the signal.
- `.BindClass(class string, on *SignalBool)`: Class is toggled based on the signal.
- `.BindAttrBool(name string, on *SignalBool)`: Boolean attribute (e.g., `disabled`, `checked`) tracks the signal.
- `.Bind(s *SignalString)`: Two-way binding for `<input>` and `<textarea>`.

### Reactive Structure
- `Show(cond *SignalBool, render func() *Element)`: Mounts/unmounts a subtree based on a condition.
- `BindChildren(s *SignalNodes)`: A container whose children track a list of nodes. Use `.Key(string)` on child elements for stable identity during reconciliation.

## 5. Void Elements
The library handles self-closing tags correctly for:
- `Input(type)`, `Img(src, alt)`, `Br()`, `Hr()` (when using builders from `tinywasm/html` or `tinywasm/image`).
These return elements with the `void` flag set, preventing the rendering of a closing tag.

## 6. Build Split Strategy
- `dom_wasm.go` & `element_wasm.go`: Implementation using `syscall/js`.
- `dom_backend.go` & `dom_stub.go` (`!wasm`): No-op / server-side logic for compilation safety.
- **WASM Memory Safety**: `Unmount` automatically releases all saved `js.FuncOf` event listeners.

## 7. LocalStorage API (WASM only)

The `dom` package provides a type-safe wrapper for the browser's `localStorage` with built-in quota management.

- `LocalStorageAvailable() bool`: Checks if storage is accessible (handles iframe sandboxes and private modes).
- `LocalStorageGet(key) (string, error)`: Retrieves a value. Returns `("", nil)` if the key is absent.
- `LocalStorageSet(key, value) error`: Persists a value. Enforces a 64KB per-value limit and a 4MB total budget to prevent crashes.
- `LocalStorageDel(key) error`: Removes a specific key.
- `LocalStorageClear() error`: Wipes all storage for the origin.

> [!NOTE]
> Quota tracking is done in-memory for performance. It assumes `dom` is the only writer for the origin during the session.

## 8. DocumentAttr API

Used to manipulate attributes on `document.documentElement` (the `<html>` tag), which is typically used for theme switching or language settings.

- `SetDocumentAttr(attr, value string)`: Sets an attribute. Passing `""` as value removes the attribute.
- `GetDocumentAttr(attr string) string`: Reads an attribute. Returns `""` if absent.

On the backend (`!wasm`), these are no-ops and return `""`, ensuring SSR safety and consistency with `GetHash()`.

## 9. Default Theme (`RootCSS`)

`dom/ssr.go` ships the default `:root { … }` theme of the framework via a single static function:

```go
//go:build !wasm

package dom

import (
	"github.com/tinywasm/css"
	_ "embed"
)

//go:embed theme.css
var rootCSS string

func RootCSS() *css.Stylesheet { return css.New(css.Raw(rootCSS)) }
```

`theme.css` is the **single source of truth** for the default tokens — colors, spacing, layout heights, dark-mode media query. There is no `CssVars` struct, no `DefaultCssVars()` constructor, no programmatic builder; the theme is plain CSS.

### Override

`dom` does not import `assetmin`. The contract is the `RootCSSProvider` interface and the free function `RootCSS`. `tinywasm/assetmin` discovers it via AST extraction during `LoadSSRModules()` and routes the result to the `open` slot of `<head>`.

Apps override the default by exposing their own `RootCSS()` from the project root's `ssr.go`. The single-override rule lives in `assetmin` (root project wins, dom is fallback, third-party modules are ignored with a warning). See [`assetmin/docs/SSR.md`](../../assetmin/docs/SSR.md).

### Distinction from `CSSProvider`

- `RootCSS()` (free function in `ssr.go`) → document-level `:root` tokens, single winner. Returns `any` (expected `*css.Stylesheet`).
- `CSSProvider.RenderCSS()` (component method) → per-component scoped styles, accumulate normally. Returns `any` (expected `*css.Stylesheet`).

These are intentionally separate: theme tokens are global and must not stack, while component styles are local and naturally compose.

## 10. Reference Mutation API

`dom.Get(id)` returns a `Reference` — a live handle to a DOM node. Use its mutation methods to update the element **in-place** without re-rendering.

> [!IMPORTANT]
> `dom.Render(parentID, comp)` calls `cleanupChildren()` before writing new `innerHTML`, which **destroys all event listeners** registered via `ref.On()`. Always prefer in-place mutation over re-rendering when you only need to change a value, attribute, or text.

| Method | JS equivalent | Use case |
|--------|---------------|----------|
| `ref.SetValue(v string)` | `element.value = v` | Reset input / textarea / select |
| `ref.SetAttr(key, value string)` | `element.setAttribute(key, value)` | Add/set attribute. Pass `""` for boolean attrs (`"disabled"`) |
| `ref.RemoveAttr(key string)` | `element.removeAttribute(key)` | Remove attribute |
| `ref.SetText(text string)` | `element.textContent = text` | Update visible text safely (no HTML parsing — XSS-safe) |

### Example: form loading state

```go
ref, _ := dom.Get("submit-btn")

// Show loading (in-place — listener survives)
ref.SetAttr("disabled", "")
ref.SetText("Enviando…")

// Restore (in-place — listener still alive)
ref.RemoveAttr("disabled")
ref.SetText("Enviar")
```

### Why not `SetInnerHTML`?

`SetText` maps to `element.textContent`, which treats the string as **plain text** — safe for user-supplied content. `innerHTML` interprets HTML and would require the caller to sanitize input. If controlled HTML injection is ever needed, a separate `SetInnerHTML` should be added with explicit XSS risk documentation.

### Backend behavior

On `!wasm` builds, all mutation methods are **no-ops** in `elementStub`. This is intentional — SSR never holds live DOM handles.

