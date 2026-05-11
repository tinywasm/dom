# PLAN — Typed CSS contract for dom

## Goal

Update `dom`'s SSR interfaces so the CSS payload is `*css.Stylesheet` instead of `string`, and expose helpers that let HTML emission and CSS emission share the same `Class` value.

## Why

Today `dom.CSSProvider` declares `RenderCSS() string`. This is the API every component implements, and the reason assetmin must parse the returned string. Changing the return type to `*css.Stylesheet` is the contract-level change that makes the entire DSL migration possible.

A second, smaller change: HTML emission needs to consume the same `css.Class` constants that the stylesheet declares. `dom` already builds attributes; it must offer an ergonomic adapter so `ClsPrimary.Attr()` produces a valid `class="btn-primary"` attribute without the caller stringifying manually.

## Interface change

`dom/interface.dom.go`:

```go
// Before
type CSSProvider interface { RenderCSS() string }

// After
import "github.com/tinywasm/css"

type CSSProvider interface { RenderCSS() *css.Stylesheet }
```

Add a sibling interface for the framework-default root:

```go
type RootCSSProvider interface { RootCSS() *css.Stylesheet }
```

## Class → attribute bridge

`css.Class` is declared in `tinywasm/css/tokens.go` without a build tag, so it is reachable from WASM code. `dom` defines the adapter:

```go
// dom.go — no build tag
package dom

import (
    "github.com/tinywasm/css"
    "github.com/tinywasm/fmt"
)

// Attr is the existing attribute type used by Element/Node builders.
// Add a constructor that accepts a css.Class so call sites read naturally:
//     dom.Button(dom.Class(css.ClsPrimary), "Save")
func Class(c css.Class) Attr { return Attr{Key: "class", Val: string(c)} }

// For multiple classes — uses tinywasm/fmt.Builder, never stdlib "strings".
// The stdlib strings package is banned across tinywasm/*: tinywasm/fmt is the
// binary-optimized replacement and must be used everywhere, including in
// SSR-only code.
func Classes(cs ...css.Class) Attr {
    b := &fmt.Builder{}
    for i, c := range cs {
        if i > 0 { b.WriteString(" ") }
        b.WriteString(string(c))
    }
    return Attr{Key: "class", Val: b.String()}
}
```

Optionally promote `css.Class.Attr()` as a convenience that returns `dom.Attr` directly — but doing so would force `tinywasm/css` to import `tinywasm/dom`, inverting the dependency. Instead, keep the adapter in `dom` only.

## Cyclic-dependency check

- `tinywasm/css/tokens.go` has no imports → safe.
- `tinywasm/dom` imports `tinywasm/css` for the type alias only → safe.
- No file in `tinywasm/css` imports `tinywasm/dom` → no cycle.

## Files modified

- `dom/interface.dom.go` — update `CSSProvider`, add `RootCSSProvider`.
- `dom/dom.go` — add `Class()` and `Classes()` attribute constructors.
- `dom/dom_backend.go` — wherever HTML rendering collects CSS from children, switch from string concatenation to `*css.Stylesheet` aggregation (or to `.String()` at the boundary).
- `dom/ssr_decoupling_test.go` — update against the new interface.
- `dom/README.md`, `dom/AGENTS.md`, `dom/docs/ARCHITECTURE.md` — document the new contract.

## Files added

- None.

## Dependency

This plan builds on the typed CSS DSL foundation already published in `tinywasm/css` (`*Stylesheet`, `.String()`, `Class`, tokens). That dependency is satisfied — no further changes to `tinywasm/css` are required for this plan.

> The keyframes-only PLAN currently sitting at `tinywasm/css/docs/PLAN.md` is **not** a dependency of this plan — `dom` neither constructs nor inspects `@keyframes`.

## Steps

1. Update `interface.dom.go` to the new signatures.
2. Add `dom.Class()` / `dom.Classes()` attribute constructors and update `Render*` paths that previously embedded class strings.
3. Update `dom_backend*.go` to collect children's CSS. **Implementation note:** the published `tinywasm/css` API does not expose a public way to merge `*css.Stylesheet` values structurally (`New(items ...item)` takes an unexported `item` type). Two options:
   - **(A) Stringify at the dom boundary:** dom calls `child.RenderCSS().String()` and concatenates. Simple; loses no information because assetmin only needs the final string.
   - **(B) Add `css.Compose(sheets ...*Stylesheet) *Stylesheet`** to `tinywasm/css` as a prerequisite, then dom aggregates structurally and assetmin calls `.String()` at the outermost boundary.

   **Decision for this plan: (A).** Rationale: assetmin already calls `.String()` on the final result; pushing the stringify one level earlier costs nothing and avoids a cross-package API change. Option (B) is a follow-up only if some consumer needs structural inspection of the merged sheet.
4. Update tests under `dom/tests/` and `ssr_decoupling_test.go`.
5. Update docs: `README.md`, `AGENTS.md`, and `docs/ARCHITECTURE.md`.

## Acceptance

- `dom.CSSProvider.RenderCSS()` returns `*css.Stylesheet`.
- A WASM build of any consumer compiles without dragging in `dsl.go` (`//go:build !wasm` in `tinywasm/css` keeps the Stylesheet machinery server-only; only `Class` and `Token` cross the boundary).
- `dom.Class(css.ClsPrimary)` produces a valid attribute and renders correctly in HTML output.
- All existing dom tests pass; new tests cover `Class()` / `Classes()` adapters.

## Out of scope

- Refactor of `dom_frontend.go` event wiring — unrelated.
- Typed HTML attribute system beyond `Class` — separate initiative.
