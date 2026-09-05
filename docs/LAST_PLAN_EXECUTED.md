---
PLAN: "feat: add Reference.ScrollIntoViewInstant for non-animated scroll-snap jumps"
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — `ScrollIntoViewInstant`, a non-smooth sibling of `ScrollIntoView`

## Context

`Reference.ScrollIntoView()` always requests `{behavior: "smooth"}` from the
browser — that JS-level option is authoritative and overrides any
`scroll-behavior` CSS on the scrolling container, so a consumer cannot get a
non-animated jump by any CSS trick from outside `dom`.

`components/calendarslider` needs exactly that: a bounded, circular strip of
month cards (‹ of the first month targets the last, › of the last targets the
first) where the WRAP transition must feel like a clean cut, not a long
smooth scroll travelling in the visually wrong direction across every month
in between. Every other (adjacent) navigation stays smooth — only the wrap
edge needs an instant jump. This plan adds the missing primitive; a separate
plan in `components` consumes it.

This is additive in BEHAVIOR — every current caller of `ScrollIntoView()`
(calendarslider, `layout/rightpanel`, `dom`'s own tests) keeps getting
exactly the same `{behavior:"smooth", inline:"start", block:"nearest"}` it
gets today. Three existing lines of code DO change (`Focus`, `ScrollIntoView`,
and `newDom`), each in a small, mechanical way spelled out below — this is
not a "pure addition, touch nothing else" plan, and does not claim to be.

**Two rules this plan follows, both from files already in this repo — read
before implementing, do not re-derive by guessing:**

1. **`dom/AGENTS.md`, "Slices Over Maps": "Maps are extremely heavy in
   TinyGo."** Not limited to the attributes/events it names as examples —
   no map literal anywhere in this package, for any reason. Building the
   options object stays exactly what `ScrollIntoView` already does —
   `<ctor>.New()` plus sequential `Set` calls, zero map machinery.
2. **`dom/AGENTS.md`, "Internal State: Singleton Fields, Not Package
   Variables": "All browser-side state lives in `domWasm` fields
   (`document`, `localStorage`, …), not in package-level variables."**
   `Focus` and `ScrollIntoView` today call `js.Global().Get("Object")`
   fresh on every invocation to get the `Object` constructor — the same
   repeated-global-lookup shape `document`/`localStorage` are already
   cached on `domWasm` to avoid. This plan adds `objectCtor` as a third
   cached field, the same way, and both methods (plus the new
   `ScrollIntoViewInstant`) read it from there instead of calling
   `js.Global()` themselves.

Verified end to end before writing this plan: applied for real against this
repo, ran `gotest` (full suite: vet, race, tests, **wasm** — standard Go
WASM toolchain — all green, coverage 55.9%, up from 53.8%, confirming the
new tests actually executed) and separately confirmed the exact
`objectCtor`/`scrollOptions` shape with a real `tinygo build -target wasm`
compile (not just `GOOS=js GOARCH=wasm go build`), then reverted before
writing this plan down. See "A pre-existing, unrelated `gotest -tinygo`
failure" below for the one thing that did NOT pass, and why it is not this
plan's problem.

## Stage 1 — cache the `Object` constructor on `domWasm`

**File: `dom_frontend.go`** — add the field next to the existing cached
globals:

```go
	document     js.Value // Cached document object
	localStorage js.Value // Cached localStorage object
	objectCtor   js.Value // Cached global Object constructor, for options objects browser APIs take (scrollIntoView, focus)
	lsUsedBytes  int      // Current localStorage budget usage in bytes (UTF-16)
```

Populate it in `newDom`, next to the other two:

```go
	return &domWasm{
		tinyDOM:      td,
		document:     js.Global().Get("document"),
		localStorage: ls,
		objectCtor:   js.Global().Get("Object"),
		lsUsedBytes:  used,
	}
```

## Stage 2 — `Focus`, `ScrollIntoView`, and the new `ScrollIntoViewInstant` read it from `e.dom`

**File: `element_wasm.go`** — `Focus` and `ScrollIntoView` currently each
call `js.Global().Get("Object").New()` themselves (lines 104-108 and
117-124 today):

```go
func (e *elementWasm) Focus() {
	opts := js.Global().Get("Object").New()
	opts.Set("preventScroll", true)
	e.val.Call("focus", opts)
}
...
func (e *elementWasm) ScrollIntoView() {
	opts := js.Global().Get("Object").New()
	opts.Set("behavior", "smooth")
	opts.Set("inline", "start")
	opts.Set("block", "nearest")
	e.val.Call("scrollIntoView", opts)
}
```

Change `Focus` to read the cached constructor (one-word change, no other
edit to that method):

```go
func (e *elementWasm) Focus() {
	opts := e.dom.objectCtor.New()
	opts.Set("preventScroll", true)
	e.val.Call("focus", opts)
}
```

Replace `ScrollIntoView`'s block with a shared helper plus both scroll
methods calling it — `ScrollIntoView`'s own behavior is identical before
and after, it just stops re-typing the object construction that
`ScrollIntoViewInstant` also needs, and reads the same cached constructor
`Focus` now does:

```go
// scrollOptions builds the options object scrollIntoView takes, off the
// singleton's cached Object constructor (domWasm.objectCtor) rather than a
// fresh js.Global() lookup — same reason document/localStorage are cached
// there. behavior is the only axis that ever varies between callers
// ("smooth" vs "instant"); inline/block stay fixed. No map — dom/AGENTS.md,
// "Slices Over Maps": three explicit Set calls instead.
func (e *elementWasm) scrollOptions(behavior string) js.Value {
	opts := e.dom.objectCtor.New()
	opts.Set("behavior", behavior)
	opts.Set("inline", "start")
	opts.Set("block", "nearest")
	return opts
}

// ScrollIntoView smooth-scrolls the element into view.
func (e *elementWasm) ScrollIntoView() {
	e.val.Call("scrollIntoView", e.scrollOptions("smooth"))
}

// ScrollIntoViewInstant jumps the element into view with no animation — an
// explicit "instant", not "auto": "auto" defers to the container's own
// scroll-behavior CSS, and callers use this method specifically to bypass
// that, unconditionally. e.g. a circular scroll-snap strip wrapping from its
// last panel back to its first, where a smooth scroll would visibly travel
// across every panel in between in the wrong apparent direction. Every
// other navigation should keep using ScrollIntoView; reach for this one
// only at the wrap boundary.
func (e *elementWasm) ScrollIntoViewInstant() {
	e.val.Call("scrollIntoView", e.scrollOptions("instant"))
}
```

`scrollOptions` is a method on `*elementWasm` (not a free function) —
`e.dom` is what carries the cached `objectCtor`, so it needs the receiver.

**File: `reference.go`** — add to the interface, directly below the
existing `ScrollIntoView` doc block:

```go
	// ScrollIntoViewInstant jumps the element into view with no animation —
	// e.g. a circular scroll-snap strip wrapping from its last panel back to
	// its first, where a smooth scroll would visibly travel across every
	// panel in between in the wrong apparent direction. Every other
	// navigation should keep using ScrollIntoView; reach for this one only
	// at the wrap boundary.
	ScrollIntoViewInstant()
```

**File: `dom_backend.go`** — add the no-op stub next to the existing one:

```go
func (e *elementStub) ScrollIntoViewInstant()                         {}
```

Match the existing stub's alignment style in that file (gofmt handles it).

## Stage 3 — test

**File: `lifecycle_wasm_test.go`** (package root, `package dom` — this test
needs the unexported `scrollOptions`, so it cannot live in `dom/tests/`
alongside the `dom_test`-package public-API tests; see `dom/AGENTS.md`'s
Testing section: "Tests requiring internal access → root of package").

Add two tests:

1. `TestScrollOptionsBehaviorDiffers` — the precise, low-level proof the
   whole plan exists for: `scrollOptions("smooth")` and
   `scrollOptions("instant")` actually differ, and neither silently
   produces the wrong one. Reuses the `scrollableItem`/`item-to-scroll`
   fixture already defined later in this same file (`TestMountedScrollableConsumer`'s
   setup) — no new fixture needed, just render it and get a real
   `*elementWasm` to call the unexported method on:

   ```go
   func TestScrollOptionsBehaviorDiffers(t *testing.T) {
   	Render("app", &scrollableItem{})
   	ref, ok := Get("item-to-scroll")
   	if !ok {
   		t.Fatal("item-to-scroll not found")
   	}
   	e, ok := ref.(*elementWasm)
   	if !ok {
   		t.Fatal("Get did not return *elementWasm")
   	}

   	smooth := e.scrollOptions("smooth")
   	if got := smooth.Get("behavior").String(); got != "smooth" {
   		t.Errorf("scrollOptions(smooth).behavior = %q, want smooth", got)
   	}
   	instant := e.scrollOptions("instant")
   	if got := instant.Get("behavior").String(); got != "instant" {
   		t.Errorf("scrollOptions(instant).behavior = %q, want instant", got)
   	}
   	for _, opts := range []js.Value{smooth, instant} {
   		if got := opts.Get("inline").String(); got != "start" {
   			t.Errorf("inline = %q, want start", got)
   		}
   		if got := opts.Get("block").String(); got != "nearest" {
   			t.Errorf("block = %q, want nearest", got)
   		}
   	}
   }
   ```

2. `TestMountedScrollableConsumerInstant` — mirrors the existing
   `deckComp`/`TestMountedScrollableConsumer` exactly (same file), as
   `deckCompInstant`, calling `el.ScrollIntoViewInstant()` instead of
   `el.ScrollIntoView()` in `Mounted()`, with its own root id
   (`"deck-scroller-instant"`, not `"deck-scroller"` — two components in one
   test binary must not share an id, see `dom.go`'s own "one id, one node"
   comment) and its own `mountedCalled`/`childFoundAtMnt` fields. This one
   is compile+call-shape coverage through the PUBLIC method, same spirit as
   the existing `ScrollIntoView` test — test 1 above is what actually
   proves the behavior difference.

Both were written and run for real against this repo (see the note at the
end of Context) — copy them as given, they already pass.

## A pre-existing, unrelated `gotest -tinygo` failure — do not try to fix it here

Running the FULL suite with `gotest -tinygo` in this repo, on `main`,
**before any change from this plan**, already crashes:

```
panic: dom: element button  is already a child of another element; ...
RuntimeError: unreachable
    ... TestChildRejectsAnElementThatAlreadyHasAParent ...
```

Confirmed by stashing every change this plan makes and re-running — byte
for byte the same crash. That test relies on `recover()` to catch and
assert on an intentional panic, and `dom/AGENTS.md`'s own "Error Handling
in TinyGo WASM" section says why that cannot work: *"`defer/recover` does
NOT work in TinyGo WASM... a panic always exits the program without running
deferred functions."* The crash aborts the whole wasm test binary, so
whether THIS plan's own new tests pass or fail is invisible to a
`gotest -tinygo` run — not because they are broken, but because the process
never reaches them. `gotest` (no `-tinygo`, the standard Go WASM toolchain,
where `recover()` works normally) is what actually exercises them, and did,
green, before this plan was written.

Do not fix `TestChildRejectsAnElementThatAlreadyHasAParent` as part of this
plan — it is a pre-existing gap unrelated to scrolling, and deserves its
own plan (rewrite it to not rely on `recover()` under TinyGo, or accept it
as a `gotest`-only, non-`-tinygo` test). Flag it to the user; do not
silently absorb it here.

## Acceptance criteria

- `grep -n "objectCtor" dom_frontend.go element_wasm.go` → the field +
  its `newDom` assignment in `dom_frontend.go`, three reads (`Focus`,
  `scrollOptions`) in `element_wasm.go`.
- `grep -n "ScrollIntoViewInstant\|func.*scrollOptions" reference.go element_wasm.go dom_backend.go` →
  `ScrollIntoViewInstant` in all 3 files, `scrollOptions` only in `element_wasm.go`.
- `grep -n "map\[string\]" element_wasm.go dom_frontend.go` → no matches.
- `grep -c "js.Global()" element_wasm.go` → 0. Every remaining
  `js.Global()` call in the package stays where it already was
  (`dom_frontend.go`'s `newDom`, `cssfeature.go`, etc.) — this plan does
  not touch those.
- `go build ./...` and plain `gotest` (standard toolchain, **not**
  `-tinygo` — see the section above) both green, coverage at or above
  55.9%.
- A standalone `tinygo build -target wasm ./...` (or an equivalent
  single-file smoke compile) succeeds — this is what actually proves the
  TinyGo-safety the map rule exists for, independent of the unrelated
  `-tinygo` test-suite crash above.
- `ScrollIntoView`'s and `Focus`'s BEHAVIOR are unchanged (still emit the
  exact same options objects as today) — their bodies are not
  byte-for-byte identical anymore.

## Stages

| Stage | File(s) | Done when |
|---|---|---|
| 1 | `dom_frontend.go` | `domWasm.objectCtor` cached in `newDom`, alongside `document`/`localStorage` |
| 2 | `element_wasm.go`, `reference.go`, `dom_backend.go` | `Focus`/`ScrollIntoView` read `e.dom.objectCtor`; `ScrollIntoViewInstant` compiles on both `wasm` and backend builds; no map literal, no leftover `js.Global()` in `element_wasm.go` |
| 3 | `lifecycle_wasm_test.go` | both new tests pass under plain `gotest`; a standalone `tinygo build` of the changed shape succeeds |
