# tinywasm/dom — Plan: Signals Engine (Fine-Grained Reactivity)

> **Master:** tinywasm/docs/PLAN.md
> **Module:** `github.com/tinywasm/dom`
> **Type:** Breaking change. Core engine + author-facing surface. **Blocks all consumer plans.**
>
> **Read first:** `docs/BINDING_MODEL.md` — explains the mental model (why signals, how bindings wire, auto-tracking). Required before implementing any signal or binding.

Goal: construction is one way (`Render()` once), and updates are surgical and first-class via
typed `Signal` bindings (no generics) — no `Update()`, no `OnMount`/`OnUpdate`, no Virtual DOM.

---

## Prerequisites

```bash
# Canonical test runner (WASM tests run against a real DOM). Required: external agents have no global gotest.
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## Development Rules

- **Documentation First:** rewrite ARCHITECTURE.md §3-4 (lifecycle/events) before code.
- **WASM only:** engine changes in `//go:build wasm` files (`dom_frontend.go`, `event_wasm.go`,
  `element_wasm.go`). Provide non-wasm stubs so signals compile for SSR (`*_backend.go`/`*_stub.go`).
- **No maps / no mutex:** WASM is single-threaded on the JS event loop; use slices for engine state.
  Use `tinywasm/fmt` for logs. `switch` not `map`; embed `Element` by value.
- **Tests:** use `gotest` (never `go test`); stdlib only (`testing`/`reflect`, no testify); dual
  WASM/stdlib via build tags sharing one runner. Publish with `gopush 'msg'`.
- **Breaking change intentional.** Remove old hooks; do not alias them.

---

## Why (rejected alternatives)

`Update()` re-renders the whole component subtree via `outerHTML`
(dom_frontend.go:264) — O(n) per change and destroys node identity
(breaks IME/focus/scroll). A VDOM is rejected by ARCHITECTURE.md:7 ("No Virtual
DOM"). Fine-grained reactivity gives O(1) surgical patches with preserved node identity and no diff.

---

## New Author Surface

```go
// Ctx is handed to Init. Register teardown for async resources (timers, websockets).
// No Refresh() needed: changing a Signal patches the DOM directly, even from a goroutine.
type Ctx interface{ OnCleanup(fn func()) }
```

`ViewRenderer.Render() *Element` stays the only required method. The optional `Init(ctx dom.Ctx)`
hook is satisfied **structurally** — the author writes the method; `dom.Ctx` is the only interface
they must name. **Removed (breaking):** `Mountable`, `Updatable`, `Unmountable`, and all author
`Update()` calls.

## API Surface — public vs internal (keep it minimal)

**Public = exactly what a component author types.** Everything the engine uses to wire/patch/clean up
stays unexported, using two Go techniques so the API is not polluted with engine plumbing:

| Public (exported) | Internal (unexported) |
|---|---|
| `SignalString`/`SignalBool`/`SignalNodes`; `NewString`/`NewBool`/`NewNodes`; `Get`/`Set`/`Toggle`/`Update` | `subscribe`/`unsub`; struct fields `v`,`subs`; `currentTracker` |
| `DeriveString`/`DeriveBool` | `subscribable` interface |
| `Ctx` (+ `OnCleanup`) | `initable` interface (the `Init` hook) |
| `Element`: `BindText(Func)`,`BindAttr(Func)`,`BindClass(Func)`,`BindAttrBool(Func)`,`Bind`,`BindChildren`,`Key`,`Autofocus` | `binding` struct, keyed-reconcile internals |
| `Show`; `SetDevMode` | `update`, `updating`, `componentByID`, `ctxFor`, cleanup registry, auto-tracking, `devMode` field |
| `Element`: `Text`, `Child`, `Attr`, `Class`, `Set(...fmt.KeyValue)` (typed; **no `Add(...any)`**) | builder internals, `children` storage |

Two enabling rules:

- **`subscribable` is unexported because its method (`subscribe`) is unexported** → only `dom`'s own
  signals can satisfy it. Auto-tracking means authors never assemble dependency lists at all.
- **`initable` is unexported but its method `Init` is exported** → the engine asserts
  `component.(initable)` while the author only writes `func Init(ctx dom.Ctx)` and never sees the
  interface (structural satisfaction).
- **`Update` becomes unexported (`update`)** — used only by `Show`/`BindChildren`. The author
  *cannot* call it, enforcing "no manual update" at compile time.

Signal struct **fields stay unexported** (`v`, `subs`); access only via `Get`/`Set`. New constants in
`dom`: none. (Downstream: unexport symbols that only one package uses — e.g. `themetoggle`'s theme
constants.)

---

## Change 1 — Typed signals (NO generics)

**No generics.** The ecosystem uses zero generic functions and follows the `tinywasm/fmt` codec rule
(fmt/codec.go:5: *"cero any, cero map"*) — typed methods per primitive. The
DOM boundary is `string`/`bool`, so signals are concrete typed cells, exactly like
`FieldWriter.String`/`Bool`. New file `signal.go` (build-tag-free; pure Go, wasm + backend):

```go
// SignalString is an observable string cell. UI text/attr/input state lives here. Explicit Get/Set.
type SignalString struct {
	v    string
	subs []func() // binding callbacks; invoked on change
}

func NewString(v string) *SignalString { return &SignalString{v: v} }
func (s *SignalString) Get() string    { return s.v }
func (s *SignalString) Set(v string) {
	if v == s.v { return }                 // equality via ==; skip no-op
	s.v = v
	for _, fn := range s.subs { fn() }     // notify only this signal's bindings — O(#bindings)
}
func (s *SignalString) Update(fn func(string) string) { s.Set(fn(s.v)) }  // read-modify-write convenience
func (s *SignalString) subscribe(fn func()) (unsub func()) { /* append; return remover */ }

// SignalBool — same shape for class/attr toggles and Show conditions.
type SignalBool struct { v bool; subs []func() }
func NewBool(v bool) *SignalBool { return &SignalBool{v: v} }
func (s *SignalBool) Get() bool  { return s.v }
func (s *SignalBool) Set(v bool) { if v == s.v { return }; s.v = v; for _, fn := range s.subs { fn() } }
func (s *SignalBool) Toggle()    { s.Set(!s.v) }                  // convenience: no Set(!Get()) noise
func (s *SignalBool) subscribe(fn func()) (unsub func()) { /* … */ }

// subscribable (UNEXPORTED) — its method is unexported, so only dom's own signals satisfy it.
type subscribable interface{ subscribe(fn func()) (unsub func()) }
```

### Auto-tracking (no explicit dependency lists)

`Get()` registers the signal with whatever reactive computation is currently running, so derived
values and function-bindings **discover their own dependencies** — the author never passes a deps
list (which was a silent footgun). Mechanism: an engine-internal `currentTracker` that `Get()` checks.

```go
// Get, with auto-tracking (engine-internal collector; invisible to authors)
func (s *SignalString) Get() string { if currentTracker != nil { currentTracker.add(s) }; return s.v }

// DeriveString / DeriveBool: read-only computed cells. Re-run automatically when any signal the
// closure READS changes — no deps argument.
func DeriveString(compute func() string) *SignalString { /* run under tracker; subscribe to reads */ }
func DeriveBool(compute func() bool) *SignalBool         { /* … */ }
```

> Numeric display values format to `string` at the component (use `SignalString`). Do **not** add
> `SignalInt`/`SignalFloat` (rejected: unnecessary complexity for a `string`/`bool` DOM boundary).
> `Update(fn)` is a **cell-level** read-modify-write (`s.Set(fn(s.v))`) — distinct from the removed
> *component* `Update()`; it never re-renders, it only computes the next value of one signal.
> Keep `subscribe` lowercase (engine-internal); authors only see `Get`/`Set`/`Toggle`/`Update`.
> **Trade-off:** auto-tracking is a small, contained "magic" (same as Solid/Vue) chosen over explicit
> deps because it is more intuitive for newcomers and correct by construction (impossible to forget a
> dependency). `DESIGN.md` records this decision.

---

## Change 2 — Bindings on `Element`

A binding records, at build time, that a DOM location tracks a signal. `Element` gets a
`bindings []binding` slice (mirrors the existing `events` slice). Any bound element gets an `id`
(like events already force, dom_frontend.go:373-376).

Builder methods (in `element.go` / `element_wasm.go`):

```go
// Raw signal bindings (the common, simplest case):
func (e *Element) BindText(s *SignalString) *Element                  // textContent tracks the signal
func (e *Element) BindAttr(name string, s *SignalString) *Element
func (e *Element) BindClass(class string, on *SignalBool) *Element
func (e *Element) BindAttrBool(name string, on *SignalBool) *Element   // disabled, checked, hidden…
func (e *Element) Bind(s *SignalString) *Element                      // two-way for <input>/<textarea>
func (e *Element) Autofocus() *Element                                // focus this node when it first appears

// Computed bindings — pass a function; auto-tracking subscribes to the signals it reads (no deps list):
func (e *Element) BindTextFunc(fn func() string) *Element
func (e *Element) BindAttrFunc(name string, fn func() string) *Element
func (e *Element) BindClassFunc(class string, fn func() bool) *Element
func (e *Element) BindAttrBoolFunc(name string, fn func() bool) *Element
```

The raw `Bind*` forms are the everyday path (bind a signal straight to a DOM spot). The `*Func` forms
cover computed values without an intermediate `Derive` or a dependency list — the engine re-runs the
closure and patches the node when any signal it read changes. (`Derive*` remains for a *named*
computed value shared across several bindings.)

`.Autofocus()` emits the `autofocus` attribute and, after the node is inserted (initial mount or when
`Show` mounts a subtree), the engine focuses it **iff** nothing else is currently focused (reuse the
`activeElement` check at dom_frontend.go:243-257) — so it never steals
focus mid-typing. Replaces the old imperative `OnMount`+`Focus()` pattern.

Wiring (post-mount, alongside `wirePendingEvents` dom_frontend.go:612):
for each binding, resolve the node by id, apply the **current** value, then
`unsub := s.subscribe(func(){ patch node })`. Record `unsub` under the owning component for cleanup.

- `renderToHTML` emits the signal's current value into the initial HTML (so first paint and SSR are
  correct), then defers the live `subscribe`.
- **`Bind` (two-way):** on `input`/`change`, `s.Set(targetValue)`. On signal→node patch, **skip if
  the node is `document.activeElement`** (user is typing) to avoid cursor jumps; otherwise set
  `value` only if different. This preserves IME composition because the node is never replaced.

### Typed builder — remove `Add(...any)` (consistent with `tinywasm/json`)

The ecosystem's house pattern is **typed methods per primitive, zero `any` in the data path** —
`tinywasm/json`'s writer is exactly this (`w.String`, `w.Int`, `w.Bool`, `w.Object`, `w.Array`; `any`
only at the I/O boundary). The DOM builder follows the same shape: **remove the generic
`Add(children ...any)`** (element.go:70) and compose through typed methods. **No new types** — reuse
`fmt.KeyValue` for attributes and `*Element` for children (avoid duplicate type declarations).

| Intent | Method | Type (reused, no new types) |
|---|---|---|
| Static text | `Text(s string)` *(exists)* | `string` |
| Nested element(s) | `Child(c ...*Element)` | `*Element` |
| Attribute | `Attr(key, val string)` *(exists)* | `string` |
| Class | `Class(c ...string)` *(exists)* | `string` |
| Pre-built attrs | `Set(kv ...fmt.KeyValue)` | `fmt.KeyValue` *(reused from `fmt`)* |
| Reactive (anything that changes) | `BindText`/`Bind`/`BindClass*`/`BindAttr*`/`BindChildren` | `*Signal*` |

- **Remove `Element.Add(...any)`.** Its three behaviors split into the typed methods above: `Set`
  absorbs the current `fmt.KeyValue` handling (class/id/attr), `Text`/`Child` cover content.
- **`tinywasm/html` migrates** its 46 variadic `Tag(children ...any)` constructors to **no-arg**
  (`Span()`, `Div()`, `Button()`…), composing via the typed methods; constructors with semantic args
  stay (`Input(type)`, `Option(value,text)`, `SelectedOption`, `Br`, `Hr`). See
  tinywasm/html/docs/PLAN.md.
- **Why now & why consistent:** matches `json`/`fmt`'s "cero any" exactly, so it is the *house* style,
  not a new one; and the breaking churn coincides with the signals migration (same call sites) so it
  is amortized — the master plan accepts breaking changes during active development.
- **Honest caveat:** typing the builder does **not** by itself stop a snapshot like
  `Text("Hola "+name.Get())`; reactivity is still guaranteed only by `BindText` requiring a signal.
  The internal `children` storage stays an implementation detail — the harness is the public API.

---

## Change 3 — Reactive structure: `SignalNodes` / `Show` (NO generics)

Lists are the one case that looked like it needed a type parameter. It does not: the DOM child slot
already holds `*Element` (`Element.children []any` + type switch). So a list is a **signal of nodes**,
not of arbitrary `T`. The component maps its data to `[]*Element` in a plain `for` loop (the way it
already builds children) and the binding reconciles by each child's key:

```go
// SignalNodes is an observable list of rendered rows. No generics; the component builds the Elements.
type SignalNodes struct { v []*Element; subs []func() }
func NewNodes(v ...*Element) *SignalNodes
func (s *SignalNodes) Get() []*Element
func (s *SignalNodes) Set(v []*Element)               // keyed reconcile: insert/remove/move changed rows only
func (s *SignalNodes) subscribe(fn func()) (unsub func())

func (e *Element) BindChildren(s *SignalNodes) *Element  // container whose children track the signal
func (e *Element) Key(k string) *Element                // stable identity for reconcile (defaults to id)

// Show mounts/unmounts a subtree when cond flips. Runs the rendered subtree's Init/cleanup.
func Show(cond *SignalBool, render func() *Element) *Element
```

`BindChildren` keeps a small `key → node` slice and does a keyed reconcile scoped to the container —
untouched rows keep their DOM identity. `Show` mounts the subtree (wiring bindings + `Init`) on
`true`, unmounts (running cleanup) on `false`. Both unsubscribe on parent unmount.

**Dev-mode key validation:** when the engine's runtime `devMode` flag is on (see Change 4),
`BindChildren`'s reconcile checks the keys of the incoming rows and `d.Log(...)`s a warning when a
key is **empty** (fell back to a volatile auto-id → reconcile can't track the row) or **duplicated**
(two rows share a key → reconcile reuses the wrong node). This surfaces the most common list bug at
the first render instead of as a mysterious mis-patch. Off by default → zero cost in production.

> Usage: `Ul().BindChildren(c.rows)` where `c.rows.Set(buildRowElements())`. No generic `For[T]`.

---

## Change 4 — Lifecycle integration & cleanup

- In `Render`/`Append` (dom_frontend.go:136, 315),
  before producing HTML: if the component satisfies the internal `initable` (has `Init(ctx Ctx)`) and
  is not yet inited, call
  `Init(ctxFor(component))` once (track inited IDs in a slice; clear on unmount so remount re-inits).
- After insertion, wire events **and** bindings (apply current value + subscribe), recording each
  `unsub` and each `OnCleanup` fn under the component ID.
- In `unmountRecursive`/`cleanupChildren` (dom_frontend.go:467,
  519): run the component's `OnCleanup` fns and call every recorded
  `unsub` (so signals don't retain dead nodes — prevents leaks).
- **Delete** the old re-mount + `OnUpdate` block in `Update`
  (dom_frontend.go:278-286) and the `OnMount` calls in
  `Render`/`Append`/`mountRecursive` (lines 191, 348, 453).
- **Unexport `Update` → `update`** (an internal primitive used by `Show`/`BindChildren` for subtree
  (re)mount when structure truly changes); authors can no longer call it. Guard against re-entrancy:

```go
// add to domWasm
updating []string
// top of update(), after resolving id:
for _, uid := range d.updating { if uid == id { d.Log("tinywasm/dom: re-entrant update on", id, "ignored"); return } }
d.updating = append(d.updating, id)
// … perform the subtree (re)mount …
// remove id from d.updating synchronously before returning — NO defer/recover (no-op in TinyGo WASM).
for i, uid := range d.updating { if uid == id { d.updating = append(d.updating[:i], d.updating[i+1:]...); break } }
```

### Runtime `devMode` flag + reactive trace (dev diagnostics)

`tinywasm/app` signals development via a **runtime** `DevMode bool` (read from the DB in
`Handler.CheckDevMode`, app/handler.go:21,92) — **not** a build tag, and it is not injected into the
WASM build. To stay consistent, `dom` exposes the same shape at runtime instead of inventing a build
tag:

```go
// on domWasm; default false. app wires it from h.DevMode when launching the WASM client.
devMode bool
func SetDevMode(on bool) { dom().devMode = on }   // public toggle, routed through the existing Log
```

When `devMode` is on, the engine emits a **reactive trace** through the existing `d.Log`: on each
notification it logs which signal patched which node, e.g.

```
[dom] estado.Set("Ocupado") → patch #btn-3 textContent
[dom] vacia.Set(false)      → unmount #empty-msg
```

This answers "who triggered this patch?" without a debugger — recovering the linear debuggability the
imperative model gave for free. The dev-mode key validation in Change 3 uses the **same** flag. When
`devMode` is false (production) both are no-ops behind a single boolean check → effectively zero cost.

### Close the construction-harness gaps (see tinywasm/docs/ARNES_DE_CONSTRUCCION.md)

The typed builder already makes the *silent* failure (plain field) uncompilable. These remaining
holes still compile but fail at runtime; downgrade each to nil-safety + a `devMode` warning so the
only failures left are compile errors or loud dev warnings:

- **Nil signal:** signal methods are **nil-safe** — `Get()` on a nil `*SignalString`/`*SignalBool`
  returns the zero value, `Set()`/`Toggle()`/`Update()` are no-ops; in `devMode` they `d.Log` a
  warning ("signal used before NewString in Init?"). A nil passed to a `BindText`/`Bind*` likewise
  warns in `devMode`. Turns a panic into a visible no-op.
- **Pointer-embedded `Element`:** in `devMode`, when mounting a component whose embedded `Element`
  is nil (the `*Element` embed mistake), `d.Log` a clear message instead of a raw `renderToHTML`
  panic.
- **`Bind` on a non-input node:** in `devMode`, warn when `.Bind(s)` targets an element that is not
  `input`/`textarea` (two-way binding has no `value` to track).

All gated by the same `devMode` flag; zero cost in production.

---

## Change 5 — Backend (SSR) parity

Signals must compile and render statically on the non-wasm path:

- `signal.go` is pure Go (no `syscall/js`) → compiles for backend unchanged; `Set`/`subscribe` work
  as plain holders.
- In `*_backend.go`, `renderToHTML` for a bound element inlines the signal's current `Get()` value
  and **ignores** the live subscription (no live DOM on the server).
- `BindChildren`/`Show` on backend render the current rows / current branch statically.

This keeps isomorphic rendering: same `Render()` produces correct SSR HTML and live WASM.

---

## Change 6 — Documentation (do FIRST, before code)

Per the documentation standard, update docs **before** writing code or running `gopush`:

1. **`interface.dom.go`** (doc comments): delete `Mountable`/`Updatable`/`Unmountable`; add
   the unexported `initable` + exported `Ctx`. Fix the `DOM.Render` comment (line 9).
2. **ARCHITECTURE.md** §"Component Lifecycle"/§"Events" (lines 82, 118, 123-129):
   replace hooks + "call `Update()`" guidance with the single contract — `Render()` once + typed
   `Signal` bindings; `Init(ctx)`; `BindChildren`/`Show`; `.Autofocus()`. Keep it abstract (what/why),
   no implementation code. Link to DESIGN.md for the rationale.
3. **`docs/DESIGN.md`** (NEW): decision record justifying the architecture so it is not re-litigated —
   the alternatives table (whole re-render vs VDOM vs **signals**); **why typed signals, no generics**
   (ecosystem convention + `tinywasm/fmt` codec rule "cero any"; DOM boundary is `string`/`bool`); and
   **why auto-tracking over explicit dependency lists** (intuitive for newcomers, correct by
   construction — accepted as contained "magic"). ARCHITECTURE.md stays clean and links here.
4. **`docs/diagrams/lifecycle.md`** (NEW): a `flowchart TD` (no `subgraph`, `<br/>` for breaks) of the
   lifecycle — `mount → Init (once) → Render → wire bindings`; `signal.Set → patch bound node`;
   `unmount → run OnCleanup + unsubscribe`. Drives the use-case tests (DDT).
5. **`README.md`**: index every file in `docs/` (PLAN.md, ARCHITECTURE.md, DESIGN.md, diagrams/);
   cross-link ARCHITECTURE↔DESIGN.

---

## Tests — frequent use cases (`gotest`)

These tests **must** exist: they lock down the everyday component-building scenarios (the ones that
broke before) and double as the canonical living examples referenced from ARCHITECTURE.md. Dual
WASM/stdlib per ssr_decoupling_test.go (shared runner, `//go:build wasm`
vs `!wasm`, single `setup_test.go`). Stdlib assertions only (`testing`/`reflect`, no testify).
File: `signal_test.go` (core, stdlib) + `lifecycle_wasm_test.go` (real DOM).

1. **Signal core (stdlib):** `Set` notifies subscribers and **skips no-op** (`==`); `Toggle` flips;
   `unsub` stops notifications; **nil-safe:** `Get()` on a nil signal returns the zero value and
   `Set`/`Toggle`/`Update` are no-ops (no panic).
2. **Counter (wasm):** state in a `SignalString`; a `.On("click")` that only `Set`s → the bound
   textContent updates and the node keeps identity (the "forgot `Update()`" class of bug).
3. **Auto-tracking (wasm):** `BindTextFunc` / `DeriveString` re-runs and patches when a signal the
   closure READS changes — **without any explicit dependency list**; reading a second signal later
   also triggers updates.
4. **Two-way input + IME (wasm):** typing `Set`s the signal; patching from the signal does **not**
   move the cursor while the input is active; the `<input>` node is never replaced.
5. **Class / attr toggle (wasm):** `BindClass`/`BindAttrBool` flip on a `SignalBool` change.
6. **Conditional `Show` (wasm):** flipping mounts/unmounts the subtree; its `Init` runs once and
   `OnCleanup` runs on unmount.
7. **Keyed list `BindChildren` (wasm):** append / remove / reorder rows patches only affected rows
   (untouched rows keep DOM identity). **Dev-mode key validation:** duplicate/empty keys emit a
   warning via `d.Log` (assert the log fires; gated behind the `dev` build tag).
8. **Load-on-init (wasm):** state `Set` in `Init` appears in the **first** rendered HTML (no flash);
   `Init` count == 1 across later structure changes.
9. **Nested component (wasm):** a child component's `Init` runs once and its subscriptions release on
   parent unmount (assert signal `subs` empty — no leak).
10. **Reentrancy guard (wasm):** an internal `Update` cascade returns without stack overflow, logs once.
11. **Dev diagnostics (wasm):** with `SetDevMode(true)`, a `Set` emits the reactive trace via the
    injected `Log`, and `BindChildren` logs on duplicate/empty keys; with `SetDevMode(false)` neither
    logs (assert against a capturing `Log`).

In-browser via tinywasm MCP on a consumer: `browser_get_errors` clean; signal changes patch the UI
with no author `Update()`.

---

## Done When

- `SignalString`/`SignalBool`/`SignalNodes`, `Get`/`Set`/`Toggle`/`Update`, `DeriveString`/`DeriveBool`,
  `BindText`/`BindClass`/`BindAttr`/`BindAttrBool`/`Bind`/`BindChildren`, `Show`,
  `Init`/`Ctx`/`OnCleanup`, `.Autofocus()` exist and are documented — **no generics**.
- Builder is typed (`Text`/`Child`/`Attr`/`Class`/`Set`), reusing `fmt.KeyValue` (no new types);
  `Add(...any)` is removed and `tinywasm/html` constructors migrated to no-arg (see its plan).
  Reactive content only via `Bind*`, which requires a signal.
- `SetDevMode` exists; when on, `BindChildren` warns on duplicate/empty keys, the engine logs the
  reactive trace, and the harness-gap warnings fire (nil signal/bind, pointer-embedded `Element`,
  `Bind` on non-input). Signal methods are nil-safe. Off by default (production no-op).
- **Construction harness:** the only ways to fail are a compile error (typed builder) or a `devMode`
  warning — no silent failures (see tinywasm/docs/ARNES_DE_CONSTRUCCION.md).
- Old hooks removed; bindings patch surgically; subscriptions released on unmount; SSR renders
  statically. Consumer plans unblocked.
- **Docs done first:** ARCHITECTURE.md rewritten, `DESIGN.md` + `docs/diagrams/lifecycle.md` created,
  `README.md` indexes all `docs/`. **Tests:** the 10 frequent-use-case tests pass under `gotest`.
