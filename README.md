# tinywasm/dom
<img src="docs/img/badges.svg">

> **Ultra-minimal DOM & reactivity toolkit for Go (TinyGo WASM-optimized).**

`tinywasm/dom` provides a type-safe, fine-grained reactive engine over the browser DOM for TinyGo/WASM. State lives in typed Signals; changing a signal patches only the bound DOM node — no Virtual DOM, no manual `Update()` calls, no re-renders.

## Features

- **Fine-Grained Reactivity**: `SignalString` / `SignalBool` / `SignalNodes` — O(1) surgical patches that preserve focus and IME composition.
- **Auto-tracking**: `BindTextFunc` / `DeriveString` discover dependencies automatically — no explicit dep lists.
- **Typed builder**: `Text`, `Child`, `Attr`, `Class`, `Set(kv ...fmt.KeyValue)` — no `Add(...any)`.
- **Two-method contract**: `Render() *Element` (pure, once per mount) + optional `Init(ctx dom.Ctx)` (side effects, once ever).
- **Keyed lists & conditional subtrees**: `BindChildren(SignalNodes)` + `Show(cond, renderFn)`.
- **No Virtual DOM**: Zero diffing; nodes are never replaced unless structure truly changes.
- **TinyGo Optimized**: Zero stdlib; `tinywasm/fmt` for logs; slices over maps; `<500KB` WASM binaries.
- **Isomorphic**: same `Render()` produces correct SSR HTML on backend and live WASM on frontend.

## Installation

```bash
go get github.com/tinywasm/dom
```

## Quick Start

```go
import (
    dom "github.com/tinywasm/dom"
    "github.com/tinywasm/html"
)

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
    )
}

func main() {
    d := dom.New(...)
    d.Render("app", &Counter{})
}
```

## Component Contract

| Method | Role | Cardinality |
|---|---|---|
| `Render() *Element` | Pure: state → structure, no side effects | Once per mount |
| `Init(ctx dom.Ctx)` | Imperative: create signals, load storage, start timers | Exactly once |

`Init` is optional — only add it when there is setup to do.

## Signals

```go
// String cell — UI text, attr, input state
name := dom.NewString("World")
name.Get()           // "World"
name.Set("Alice")    // notifies all bindings
name.Update(func(v string) string { return v + "!" })

// Bool cell — class/attr toggles, Show conditions
active := dom.NewBool(false)
active.Toggle()

// List of rendered rows — keyed reconcile
rows := dom.NewNodes(elem1, elem2)
rows.Set(newRows)

// Derived (auto-tracking — no deps list)
full := dom.DeriveString(func() string { return first.Get() + " " + last.Get() })
```

## Element Builder

```go
html.Div().
    Class("card").
    Attr("role", "region").
    Child(
        html.Span().BindText(name),
        html.Input("text").Bind(name),           // two-way
        html.Button("Save").BindAttrBool("disabled", saving),
    )
```

Binding methods:

| Method | DOM target |
|---|---|
| `.BindText(s *SignalString)` | `textContent` |
| `.BindAttr(name, s)` | attribute value |
| `.BindClass(class, on)` | class toggle |
| `.BindAttrBool(name, on)` | boolean attribute (`disabled`, `checked`…) |
| `.Bind(s)` | two-way `<input>`/`<textarea>` |
| `.BindChildren(s *SignalNodes)` | keyed child list |
| `.BindTextFunc(fn)` | computed text (auto-tracking) |
| `.Autofocus()` | focus on first appearance |

Structural:

```go
dom.Show(visible, func() *dom.Element { return html.Div(...) })  // mount/unmount subtree
html.Ul().BindChildren(c.rows)                                    // keyed list
```

## Lifecycle

```
Init (once) → Render → wire bindings & events
signal.Set  → patch bound node (O(1))
unmount     → run OnCleanup + unsubscribe signals
```

## Mount Point

Always `"app"`, never `"body"` — `Render("body", ...)` overwrites `innerHTML` and destroys the SVG sprite injected by `tinywasm/assetmin`.

## Dev Mode

```go
dom.SetDevMode(true) // enabled at runtime; default false (production no-op)
```

When on:
- Reactive trace: logs `signal.Set → patch #node-id`
- `BindChildren` warns on duplicate/empty keys
- Nil signal / non-input `.Bind` / pointer-embedded `Element` emit warnings instead of panicking

## Related Packages

- [tinywasm/html](https://github.com/tinywasm/html) — HTML element builders (no-arg: `Div()`, `Span()`, `Button()`…)
- [tinywasm/svg](https://github.com/tinywasm/svg) — SVG builders + icon sprite
- [tinywasm/image](https://github.com/tinywasm/image) — Image element builders

## Documentation

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — component model, builder API, lifecycle
- [docs/DESIGN.md](docs/DESIGN.md) — decision record: why signals, no generics, auto-tracking
- [docs/BINDING_MODEL.md](docs/BINDING_MODEL.md) — mental model with worked examples
- [docs/diagrams/lifecycle.md](docs/diagrams/lifecycle.md) — Mermaid lifecycle flowchart
- [docs/TRADEOFFS.md](docs/TRADEOFFS.md) — fine-grained reactivity vs VDOM trade-offs
- [AGENTS.md](AGENTS.md) — constraints for agents and contributors

## License

MIT
