# PLAN: Fix Child Component Event Listeners Lost After Parent Update()

> **Status:** Pending  
> **Priority:** P1  
> **Affects:** `dom_frontend.go` — `Update()` and `wirePendingEvents()`

## Library Philosophy (must be respected in the fix)

Any change to `tinywasm/dom` must follow these constraints:

- **Zero stdlib**: Never import `fmt`, `strings`, `errors`, `strconv`. Use `github.com/tinywasm/fmt`.
- **Slices over maps**: Use `[]T` instead of `map[K]V` — maps are heavily heap-allocated in TinyGo.
- **Value embedding**: Components embed `dom.Element` as a value (`MyComp struct { dom.Element }`), never as a pointer (`*dom.Element`). One allocation instead of two; no nil-panic risk.
- **No VDOM**: State changes call `.Update()` directly — no diffing, no intermediate tree.
- **TinyGo-safe**: No goroutines with shared mutable state, no `sync.Mutex`, no `interface{}` boxing beyond what `js.Value` requires.
- **Minimal allocations**: Prefer stack values. Avoid closures that capture large structs. Reuse slices with `[:0]` reset instead of re-allocating.
- **Isomorphic**: `dom_frontend.go` (`//go:build wasm`) and `dom_backend.go` (`//go:build !wasm`) must implement the same interface — the fix must compile cleanly for both targets.

## Bug Description

When a parent component calls `Update()`, child component event listeners wired in `OnMount()` are
not correctly re-wired after the re-render. The result is that interactive child components (e.g.
a search input inside a `SelectSearch`) stop responding to user input after the parent updates.

### Observed symptom

- `App` holds a `SelectSearch` as a struct field (`App.ss SelectSearch`).
- On first render: `SelectSearch.OnMount()` wires `input` and `click` handlers correctly.
- After `App.Update()` is called (e.g. from `OnSelect`): the search input no longer filters options.

### Root cause (suspected — `dom_frontend.go`)

1. `Update(component)` calls `cleanupListeners(id)` with the **parent** ID only — child listeners
   registered under their own ID are not cleaned up.
2. `wirePendingEvents()` runs with `d.currentComponentID = parentID`, so any pending child events
   are attributed to the parent's ID bucket, not the child's.
3. Listeners wired directly via `dom.Get(id+"-search").On(...)` inside `OnMount()` are not tracked
   in `pendingEvents` at all — they go directly to the real DOM element and survive as stale
   closures after the element is replaced by `outerHTML`.

Relevant lines in `dom_frontend.go`:
- `Update()` L178-180: `cleanupChildren(id)` + `cleanupListeners(id)` — parent scope only.
- `Update()` L211-214: `d.currentComponentID = id` before `wirePendingEvents()` — wrong scope for children.
- `mountRecursive()` L359-376: calls `OnMount()` on each child — correct, but stale DOM closures
  from the previous render may conflict.

## Fix Plan

### Step 1 — Audit listener tracking for children

In `Update()`, after collecting `children` from `renderToHTML`, clean up each child's listeners
before calling `mountRecursive`:

```go
for _, child := range children {
    d.cleanupListeners(child.GetID())
}
```

### Step 2 — Scope `currentComponentID` per child during `wirePendingEvents`

When wiring events for child components, set `currentComponentID` to the child's ID, not the
parent's. Consider passing the target ID into `wirePendingEvents` or running it per-component.

### Step 3 — Track `OnMount`-wired listeners

Listeners registered via `dom.Get(...).On(...)` inside `OnMount()` bypass `pendingEvents`.
Options:
- a) Make `elementWasm.On()` push to `pendingEvents` using `currentComponentID` at call time.
- b) Document that `OnMount` listeners must use `el.On()` (already on the element ref) and that
  the element ref is stale after re-render — callers must re-fetch via `dom.Get`.

Option (b) is a documentation fix only and does not fully solve the problem.
Option (a) is the correct fix but requires verifying that `currentComponentID` is correctly set
during `OnMount()` execution (it is — see `mountRecursive` L361).

## Testing

Install `gotest` to run browser-emulated WASM tests (required — standard `go test` cannot run `//go:build wasm` code):

```bash
go install github.com/tinywasm/gotest@latest
```

## Reproducer

Test added at `test/uc_child_listeners_test.go` — `TestChildListenersAfterParentUpdate`.

Confirmed failing with `gotest`:
```
FAIL: TestChildListenersAfterParentUpdate
    after parent Update: expected 2 input events, got 1 — listener lost after parent re-render
```

Run with:
```bash
gotest ./test/... -run TestChildListenersAfterParentUpdate -v
```

## Acceptance Criteria

- [ ] `TestChildListenersAfterParentUpdate` passes with `gotest`.
- [ ] Typing in `SelectSearch` search input filters options after parent `Update()`.
- [ ] No regression in existing `mountRecursive` / `cleanupListeners` tests.
