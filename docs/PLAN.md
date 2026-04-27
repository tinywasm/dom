# PLAN: OnMount Listeners Lost After Component Self-Update

> **Status:** Pending  
> **Priority:** P1  
> **Affects:** `dom_frontend.go` — `Update()`

## Library Philosophy (must be respected in the fix)

- **Zero stdlib**: Never import `fmt`, `strings`, `errors`, `strconv`. Use `github.com/tinywasm/fmt`.
- **Slices over maps**: Use `[]T` instead of `map[K]V`.
- **Value embedding**: Components embed `dom.Element` as a value, never as a pointer.
- **TinyGo-safe**: No goroutines with shared mutable state, no `sync.Mutex`.
- **Isomorphic**: fix must compile for both `wasm` and `!wasm` targets.

## Bug Description

When a component calls `c.Update()` from within a listener wired in `OnMount()`, all
`OnMount`-wired listeners are permanently lost after the first update.

Reproduced in `selectsearch.SelectSearch`:
- `OnMount()` wires `input` handler on `#id-search` and `click` handler on `#id-options`.
- User clicks an option → `c.Update()` is called inside the click handler.
- After the update, neither the search input nor the options click respond.

## Root Cause

In `Update()` (`dom_frontend.go`):

```
cleanupListeners(id)       ← removes ALL tracked listeners for this component ✓
re-render → outerHTML      ← DOM replaced ✓
wirePendingEvents()        ← re-wires only static events from Render() ✓
OnUpdate() if implemented  ← called if component implements Updatable ✓
mountRecursive(children)   ← mounts NEW children only
```

**Missing**: `OnMount()` is never called on the component being updated itself.  
After `Update()`, the dynamic listeners wired in `OnMount()` are gone and never re-wired.

The same applies transitively to parent updates: if a parent calls `Update()`, child
components that wired listeners in `OnMount()` also lose them (see `CHECK_PLAN.md`).

## Observed Symptom (via MCP browser)

1. SelectSearch renders and dropdown opens — ✓ works.
2. User clicks "Apple" (first selection) → `c.Update()` fires → "Selected ID: 1" shows — ✓ works.
3. User opens dropdown again and clicks "Banana" → no response — ✗ bug.
4. User types in search box after any selection → no filtering — ✗ bug.

## Fix Plan

### Step 1 — Call `OnMount()` on the component after self-update

In `Update()`, after `wirePendingEvents()` and `OnUpdate()`, re-mount the component itself:

```go
// Re-wire OnMount listeners — DOM was replaced, listeners must be re-registered.
prevID = d.currentComponentID
d.currentComponentID = id
if mountable, ok := component.(Mountable); ok {
    mountable.OnMount()
}
d.currentComponentID = prevID
```

This is safe because `cleanupListeners(id)` ran first, so no duplicate listeners accumulate.

### Step 2 — Verify child scope is preserved

Ensure the `currentComponentID` is correctly restored after the OnMount call so
subsequent `mountRecursive(child)` calls use the correct child IDs (not the parent's).

## Testing

`gotest` compila y ejecuta los tests WASM automáticamente — el agente no necesita
manejar TinyGo, `wasm_exec.js`, ni ningún toolchain de browser. Basta con instalarlo
una vez y ejecutarlo:

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

Reproducir el bug (fast path — solo el test específico):

```bash
gotest -run TestSelfUpdateRewiresOnMountListeners
```

Verificar sin regresiones (full suite — vet + race + stdlib + wasm):

```bash
gotest
```

## Acceptance Criteria

- [ ] `TestSelfUpdateRewiresOnMountListeners` passes with `gotest`.
- [ ] `TestChildListenersAfterParentUpdate` still passes (no regression — see `CHECK_PLAN.md`).
- [ ] SelectSearch: clicking an option updates the header AND subsequent clicks/searches still work.
- [ ] No duplicate listener registration (verify by clicking the same option 3× — `SelectFired` must equal click count, not multiply).
- [ ] No regression in `gotest ./test/...`.
