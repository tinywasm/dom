# tinywasm/dom — Plan: Default theme via `RootCSS()`

## Problem

`dom/ssr.go` exposes the default theme via `(CssVars) RenderCSS()` which returns a computed string. Any external static extractor walking the AST cannot evaluate function calls — the extracted CSS is empty and the default theme never reaches the document `<head>`.

The `CssVars` type, `DefaultCssVars()`, and `(CssVars) renderCSS()` exist only to render `:root { … }` from struct fields. The same values are also hand-maintained in `theme.css`. Two parallel sources of truth for the same tokens.

## Decision

`dom` ships **one** theme: the static `theme.css` file, exposed by **one** function.

```go
//go:build !wasm

package dom

import _ "embed"

//go:embed theme.css
var rootCSS string

// RootCSS returns the default `:root { … }` theme as a CSS string.
// Static extractors (assetmin) read it via the `//go:embed` directive
// during AST walking. Apps that want a custom theme expose their own
// `RootCSS()` from the project root — assetmin resolves the override.
func RootCSS() string { return rootCSS }
```

Everything else is removed.

`dom` does not import `assetmin`. The contract is purely the function name `RootCSS`. assetmin discovers it by AST inspection (covered in `assetmin/docs/PLAN.md`).

## Changes

### 1. `dom/ssr.go` — replace contents

Replace the file entirely with the snippet above. No `CssVars`, no method, no second function.

### 2. `dom/ssr.theme.go` — delete

Removes:

- `var ThemeCSS string` (public embed)
- `type CssVars struct { … }`
- `func DefaultCssVars() CssVars`
- `func (c CssVars) renderCSS() string`

The single source of truth for the default theme becomes `theme.css`.

### 3. `dom/ssr.theme_test.go` — delete

Replaced by `dom/ssr_test.go` (see Tests section).

### 4. `dom/interface.dom.go` — unchanged

The `CSSProvider interface { RenderCSS() string }` at line 79 stays. It is the runtime contract for **components** that ship their own CSS (e.g., a widget shipping `.my-button { … }`) — orthogonal to the document-level `:root` theme. Verify no theme-related code depends on this interface; no change expected.

### 5. `dom/theme.css` — unchanged

Keep as-is. It is now the only source of theme tokens.

## Tests

All under `dom/`. Run with `go test ./...` from the package root.

### `dom/ssr_test.go` (new — replaces `ssr.theme_test.go`)

| Test | Setup | Assertion |
|---|---|---|
| `TestRootCSS_NotEmpty` | call `RootCSS()` | result is non-empty |
| `TestRootCSS_ContainsRootSelector` | call `RootCSS()` | result contains `":root"` |
| `TestRootCSS_ContainsCoreToken` | call `RootCSS()` | result contains `"--mag-cua"` (proves the embed read the real `theme.css`, not a stub) |
| `TestRootCSS_ContainsDarkModeQuery` | call `RootCSS()` | result contains `"@media (prefers-color-scheme: dark)"` |
| `TestRootCSS_DoesNotUseHighSpecificity` | call `RootCSS()` | result does NOT contain `:root:not([data-theme="light"])` (regression guard from a prior fix) |
| `TestRootCSS_AstShape` | parse `ssr.go` with `go/parser`, find `FuncDecl` named `RootCSS`, inspect its return | the return is an `*ast.Ident` whose name appears as a `var` in the file with a `//go:embed theme.css` doc comment. Locks in the AST shape that assetmin relies on — if someone changes `RootCSS()` to return a function call or other unparseable expression, this test fails before extraction silently breaks. |

### `dom/ssr_decoupling_test.go` (new)

| Test | Setup | Assertion |
|---|---|---|
| `TestDomDoesNotImportAssetmin` | walk all `*.go` files in `dom/` (recursively, including build-tagged), parse imports | no file imports any package whose path contains `assetmin`. Locks in the one-way coupling rule. |

### Removed tests

- `TestCssVars_Render` and its subtests — the type no longer exists.
- `TestThemeCSS_Embedded` — `ThemeCSS` no longer exists; `TestRootCSS_*` covers the embed.

## Order of implementation

1. Delete `dom/ssr.theme.go`.
2. Delete `dom/ssr.theme_test.go`.
3. Replace `dom/ssr.go` with the new contents.
4. Add `dom/ssr_test.go` with the table above.
5. Add `dom/ssr_decoupling_test.go`.
6. `go build ./...` and `go test ./...` from `dom/` clean.
7. Run a tree-wide grep for `dom.CssVars`, `dom.DefaultCssVars`, `dom.ThemeCSS` to catch any unexpected consumer; resolve before merging.

## Breaking changes

| Removed symbol | Replacement | Mitigation |
|---|---|---|
| `dom.ThemeCSS` (var) | `dom.RootCSS()` | grep confirmed the only consumer was the deleted test |
| `dom.CssVars` (type) | none — apps write their own `theme.css` and expose it via their own `RootCSS()` | grep tree-wide; document migration in commit message if a consumer is found |
| `dom.DefaultCssVars()` | none — token defaults are now expressed only in `theme.css` | same as above |
| `(CssVars).RenderCSS()` (method) | none | same as above |

If a consumer is found that legitimately needs a programmatic theme builder, that is a separate package's concern (e.g., a future `tinywasm/themekit`), not dom's.

## Out of scope

- How assetmin discovers, extracts, and routes `RootCSS()` into the `<head>` — covered in `assetmin/docs/PLAN.md`.
- A programmatic theme-builder API. Apps that need one build it themselves with `text/template` over their own CSS file, then expose the result via their root project's `RootCSS()`.
