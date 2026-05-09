# Agent Guide — `tinywasm/dom`

Constraints and rules for agents adding or modifying functionality in this package.
Read this before making any change.

---

## Fundamental Constraint

**`tinywasm/dom` is the only package in the tinywasm ecosystem that may import `syscall/js`.**

Any other package (`tinywasm/components/*`, apps, etc.) that needs browser APIs must call public functions from `dom`. Never add `syscall/js` imports outside this package.

---

## Package Pattern: Free Functions, No Types

The established pattern is **free public functions**, not methods on a custom type.

```
✅ SetHash(hash)     LocalStorageGet(key)     SetDocumentAttr(attr, value)
❌ h.SetHash(hash)   ls.Get(key)              doc.SetAttr(attr, value)
```

- No `Handle` type. No receiver types wrapping DOM state.
- No sub-packages inside `dom`.
- New APIs must follow the same flat function style as `SetHash`/`GetHash`, `LocalStorageGet`/`LocalStorageSet`, `SetDocumentAttr`/`GetDocumentAttr`.

---

## Build Split Rules

| API category | `//go:build wasm` file | `//go:build !wasm` stub |
|---|---|---|
| Called only from `*_wasm.go` / WASM-only callers | ✅ Required | ❌ None — omit the stub |
| Called from code without a build tag (e.g. inside `Render()`) | ✅ Required | ✅ Required — no-op / `""` |

**Examples:**
- `LocalStorage*` → only called from WASM code → **no backend stub**.
- `SetDocumentAttr`, `GetDocumentAttr` → called from `ThemeSwitch.Render()` (no build tag) → **both files required**.
- Backend stubs return `""` or are no-ops. Never keep in-memory state in a backend stub — concurrent server requests would share it and contaminate each other.

---

## Error Handling in TinyGo WASM

**`defer/recover` does NOT work in TinyGo WASM.**

> On architectures where `recover` is not implemented, a panic always exits the program without running deferred functions.

Use O(1) guards instead:

```go
// ✅ Guard before any JS call
if !LocalStorageAvailable() {
    return "", Err("localStorage unavailable")
}

// ❌ Never rely on defer/recover to catch JS panics
defer func() { recover() }()  // silently does nothing in TinyGo
```

For quota management, maintain an in-memory counter (see `lsUsedBytes` in `domWasm`) and validate before calling JS, not after.

---

## Zero Standard Library

Never import `fmt`, `strings`, `errors`, or other stdlib packages.
Use `github.com/tinywasm/fmt` for formatting and error construction:

```go
import . "github.com/tinywasm/fmt"

return Err("localStorage unavailable")
return Errf("value too large for key %s", key)
```

---

## Slices Over Maps

Maps are extremely heavy in TinyGo. Use `[]fmt.KeyValue` for attributes and events — not `map[string]string`.

---

## The `""` = Absent Convention

Throughout the package, an empty string means "absent" or "remove":

- `LocalStorageGet` returns `("", nil)` for a missing key (not an error).
- `SetDocumentAttr(attr, "")` removes the attribute.
- `GetDocumentAttr` returns `""` if the attribute is absent.
- `GetHash()` backend returns `""` — server has no URL state.

New APIs must follow this same convention.

---

## Internal State: Singleton Fields, Not Package Variables

All browser-side state lives in `domWasm` fields (`document`, `localStorage`, `lsUsedBytes`), not in package-level variables. This keeps browser state centralized in the singleton and avoids shared mutable globals.

---

## DOM Boundaries

`dom` is a **JS bridge** — it exposes DOM primitives without business logic.

- `SetDocumentAttr("data-theme", "dark")` is the same primitive as `SetDocumentAttr("lang", "es")`. Semantic meaning (which values are valid, when to rotate them) lives in the component that calls it.
- Do not add theme logic, routing logic, or form logic to `dom`. Those belong in `themeswitch`, `router`, `form`, etc.

---

## Adding a New Browser API

Checklist before opening a PR:

1. **Does it require `syscall/js`?** → it belongs in `dom`. If not, reconsider the location.
2. **Is it called from tag-less code?** → add a `!wasm` stub (no-op / `""`).
3. **Does it only run in the browser?** → no stub needed; mark the file `//go:build wasm`.
4. **Error path**: use O(1) guards, no `defer/recover`.
5. **Naming**: full words, no abbreviations (`LocalStorageDel` not `LsDel`).
6. **No new types or sub-packages.**
7. **Tests**: real browser tests in `dom/tests/uc_<name>_test.go` (`//go:build wasm`, `package dom_test`). Run with `gotest`.

---

## Testing

Tests that exercise browser APIs run in a real browser via `gotest`:

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
gotest
```

- Public API tests → `dom/tests/uc_*_test.go` (`package dom_test`)
- Tests requiring internal access → root of package (`package dom`)
- All new browser API tests go in `dom/tests/`, not in the package root.
- Each test cleans up after itself: call `LocalStorageClear()` and `SetDocumentAttr("data-theme", "")` in cleanup, ignoring the returned error.

---

## Mount Points

```go
✅ Render("app", &App{})    // correct — sprite SVG stays intact
❌ Render("body", &App{})   // destroys the SVG sprite injected by assetmin
```

---

## Events and Lifecycle

- Attach event handlers inside `Render()`, not inside `OnMount()`.
- `OnMount()` is reserved for third-party JS integration or DOM geometry measurement.
- Embed `dom.Element` **as a value**, never as a pointer.

```go
✅ type Counter struct { dom.Element; count int }
❌ type Counter struct { *dom.Element; count int }
```

---

## Reference

- `docs/ARCHITECTURE.md` — full API reference, build split, rendering patterns.
- `web/client.go` — canonical usage example.
