# PLAN: Child Component Element Lost in DOM After Parent+Self Update

> **Status:** Pending  
> **Priority:** P1  
> **Affects:** `dom_frontend.go` — `renderToHTML()` (Component case) and `Update()`  
> **Supersedes:** previous self-update fix (which was necessary but not sufficient).

## Library Philosophy (must be respected in the fix)

- **Zero stdlib**: Never import `fmt`, `strings`, `errors`, `strconv`. Use `github.com/tinywasm/fmt`.
- **Slices over maps**: Use `[]T` instead of `map[K]V`.
- **Value embedding**: Components embed `dom.Element` as a value, never as a pointer.
- **TinyGo-safe**: No goroutines with shared mutable state, no `sync.Mutex`.
- **Isomorphic**: fix must compile for both `wasm` and `!wasm` targets.

## Bug Description

Real-world reproduction: `tinywasm/components/selectsearch`.

When a child component's `OnMount` listener performs **parent.Update() THEN c.Update()**
(both within the same handler), the child's OnMount listeners are permanently lost
after the first interaction.

`gotest` output (test reproducer below):

```
--- FAIL: TestParentThenSelfUpdate (0.00s)
    tinywasm/dom: component element not found during Update: 3
    second click: expected SelectFired=2, got 1 — click listener lost after parent+self update
    search after clicks: expected InputFired=1, got 0 — input listener lost
    third click: expected SelectFired=3, got 1
```

The diagnostic log `component element not found during Update: 3` is the smoking gun:
the child's element with `id=3` is **not in the DOM** when `c.Update()` runs.

## Root Cause (real)

There are **two combined defects**:

### Defect A — `renderToHTML` does not inject component ID on nested children

In [dom_frontend.go:293-362](dom/dom_frontend.go#L293-L362), the `Component` case
walks `vr.Render()` (the child's render tree) but never calls `injectComponentID`
on the resulting root element:

```go
case Component:
    *comps = append(*comps, v)
    var childID string
    if v != nil {
        if v.GetID() == "" { v.SetID(generateID()) }
        childID = v.GetID()
    }
    if vr, ok := v.(ViewRenderer); ok {
        s += d.renderToHTML(vr.Render(), comps, childID)  // ← childID never injected on root!
    }
```

Compare with [dom_frontend.go:130-132](dom/dom_frontend.go#L130-L132), the **outer**
`Render()` path does inject:

```go
root := vr.Render()
injectComponentID(root, component.GetID())   // ← only here
html = d.renderToHTML(root, &children, component.GetID())
```

Effect: a child component's outer DOM element has **no `id` attribute**, even
though `child.GetID()` returns a valid string (e.g. `"3"`). `getElementById("3")`
returns `null`.

The first interaction works because OnMount uses **inner** IDs (`id+"-search"`,
`id+"-options"`) which are explicitly set by the component author. The bug
only surfaces when `c.Update()` looks up the **outer** ID via `getElementById`.

### Defect B — `Update()` returns early after `cleanupListeners`

In [dom_frontend.go:186-210](dom/dom_frontend.go#L186-L210):

```go
d.cleanupChildren(id)
d.cleanupListeners(id)        // ← removes listeners FIRST
// ...renders HTML...
elRaw := d.document.Call("getElementById", id)
if elRaw.IsNull() || elRaw.IsUndefined() {
    d.Log("tinywasm/dom: component element not found during Update:", id)
    return                    // ← returns early — no re-mount!
}
```

When Defect A causes `getElementById(id)` to return null:
1. Listeners are already removed.
2. Update returns early without re-rendering or calling OnMount.
3. The component is left with no listeners until the next parent update.

## Sequence Diagram (from test)

```
 Initial state: parent="2", child="3"
 DOM: <div id="2"><div>          ← child outer div has NO id!
        <input id="3-search"/>
        <div id="3-options"><div id="3-opt-a"/></div>
      </div></div>

 Click on #3-opt-a:
   ├─ #3-options click handler fires (v1)
   │   ├─ child.OnSelect()
   │   │    └─ parent.Update()
   │   │         ├─ cleanupChildren(2) → unmount child, remove v1 listeners
   │   │         ├─ getElementById("2") ✓ → set outerHTML
   │   │         └─ mountRecursive(child) → register v2 listeners
   │   │              (still on inner #3-search and #3-options)
   │   └─ child.Update()
   │         ├─ cleanupListeners("3") → removes v2 listeners ⚠
   │         ├─ getElementById("3") ✗ NULL — child's outer div has no id!
   │         └─ EARLY RETURN — no re-mount, no listeners

 Result: zero listeners on the child. Subsequent clicks/inputs are dead.
```

## Fix Plan

### Step 1 — Inject component ID on nested children in `renderToHTML`

In `dom_frontend.go`, the `Component` case must call `injectComponentID` on
the rendered root before walking it (mirroring the outer `Render()`/`Update()`
contract):

```go
case Component:
    *comps = append(*comps, v)
    var childID string
    if v != nil {
        if v.GetID() == "" { v.SetID(generateID()) }
        childID = v.GetID()
    }
    if vr, ok := v.(ViewRenderer); ok {
        root := vr.Render()
        injectComponentID(root, childID)               // ← FIX
        s += d.renderToHTML(root, comps, childID)
    } else if en, ok := v.(elementNode); ok {
        root := en.AsElement()
        injectComponentID(root, childID)               // ← FIX
        s += d.renderToHTML(root, comps, childID)
    } else if el, ok := v.(*Element); ok {
        injectComponentID(el, childID)                 // ← FIX
        s += d.renderToHTML(el, comps, childID)
    } else {
        s += v.RenderHTML()
    }
```

### Step 2 — Verify `Update()` self-OnMount fix is preserved

The previous fix (call `OnMount()` on the updated component itself) remains
necessary so that listeners survive a self-update — keep it as already
implemented.

### Step 3 — Optional defensive log

Consider downgrading the `component element not found during Update` log to
explicitly note that this indicates **Defect A** (component root has no id),
not a transient timing issue, so future debugging is faster.

## Test Reproducer

Already added at [test/uc_self_update_test.go](dom/test/uc_self_update_test.go):

- `TestSelfUpdateRewiresOnMountListeners` — single-component self-update path
  (currently passes because of the previous OnMount fix).
- `TestParentThenSelfUpdate` — full SelectSearch flow: handler invokes
  `parent.Update()` then `child.Update()`. **Currently fails** with the
  exact error seen in production:

```
--- FAIL: TestParentThenSelfUpdate
    tinywasm/dom: component element not found during Update: 3
    second click: expected SelectFired=2, got 1
    search after clicks: expected InputFired=1, got 0
    third click: expected SelectFired=3, got 1
```

## Testing

`gotest` automatically compiles and runs WASM tests — the agent does not need
to deal with TinyGo, `wasm_exec.js`, or any browser toolchain. Install once and
run:

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

Reproduce the bug:

```bash
gotest -run TestParentThenSelfUpdate
```

Verify no regressions (full suite — vet + race + stdlib + wasm):

```bash
gotest
```

## Acceptance Criteria

- [ ] `TestParentThenSelfUpdate` passes (no `component element not found` log).
- [ ] `TestSelfUpdateRewiresOnMountListeners` still passes.
- [ ] `TestChildListenersAfterParentUpdate` still passes.
- [ ] Live SelectSearch demo: select multiple options consecutively + search
      filtering works after every selection.
- [ ] No regression in `gotest` (full suite passes including `wasm`).
