---
PLAN: "feat!: Show builds once and toggles display — the re-attach panic becomes unrepresentable"
TAG: v0.13.0
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 7054643587315192779
PR: https://github.com/tinywasm/dom/pull/20
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
>
> **Stage S1 of a 3-repo change.** Master plan:
> `app-releases/docs/SHOW_HARNESS_MASTER_PLAN.md`.

# Plan — `dom`: close the `Show` harness (breaking)

## Why

Production panic reported in `tinywasm/layout` (`docs/BUG_DOM.md`), reproduced live in the
`platformd` demo and reproduced **in this repo** by `show_regression_wasm_test.go`
(already on main, red today — run `gotest -run TestShowSecondToggleSharedContent`):

```
dom: element div shared-body is already a child of another element; one element has one
parent — build a second instance instead of sharing this one
```

Mechanism: `Show(cond, render func() *Element)` re-invokes `render` on **every**
false→true transition. The signature says nothing about it, so every consumer closes over
`*Element`s built outside the callback (modaldialog's `modalContent`, selectsearch's
`searchInput`/`optList` — 2 of 2 consumers in the ecosystem fell into the trap). The first
open attaches the captured element; the second open calls `Child` on it again and panics.
In production the panic runs on the js event goroutine and **kills the whole Go/WASM
program**.

This is a harness hole, not a consumer mistake (see
`https://github.com/tinywasm/layout/blob/main/docs/CONSTRUCTION_HARNESS.md`):

- **"Things you have to remember"** — "the callback must return a completely fresh tree on
  every invocation" is exactly that.
- **Invariants checked at runtime** — the failure is a runtime panic, and only on the
  *second* toggle, so it survives every first-use test.
- **Fail at compile time, not at runtime** — the fix must make the illegal state
  *unrepresentable*, not document it.

## The change (breaking — no back-compat on purpose)

New signature, both in `dom_frontend.go` (`//go:build wasm`) and `dom_backend.go`
(`//go:build !wasm`):

```go
func Show(cond *SignalBool, content Component) *Element
```

**The callback is deleted.** There is no builder to re-run, so the "second attach" state
cannot be written. New semantics:

- The subtree is `Child`ed into the container **once** at construction — the `attached`
  flag is set exactly once, by construction.
- The subtree mounts **once** with the parent: serialized into the parent's HTML, bindings
  wired, `Init` runs once. This aligns with the Component Contract ("the framework runs
  Init ONCE") — the old per-toggle mount/cleanup contradicted it.
- Visibility is toggled with the inline style `display:none` on the **container** (a bare,
  dom-owned div — no author CSS can accidentally override an inline style, unlike the
  `hidden` attribute, which any `display:flex` rule beats).
- Node identity, event listeners, signal bindings, focus and scroll state survive toggles.
  Bindings keep patching while hidden (nodes are styled, not detached), so content bound
  to signals is current the moment it reappears.
- SSR emits the child always, with `style='display:none'` on the container when `cond` is
  false — WASM and SSR initial markup agree by construction.

---

## Rules of this repository (read before writing code)

- **No Go stdlib in shared (untagged) or WASM files.** Use `github.com/tinywasm/fmt`.
  `_test.go` files are exempt (stdlib `testing`/`strings` allowed — do NOT "fix" them).
- **Free functions, no types.** `Show` stays a package-level function.
- **`dom` is the only package that may import `syscall/js`.**
- **Build split:** an API called from untagged code (consumers call `Show` inside
  `Render()`) needs BOTH the `wasm` implementation and the `!wasm` stub, with identical
  signatures.
- **Testing:** `go install github.com/tinywasm/devflow/cmd/gotest@latest` first, then run
  `gotest` (never `go test`). It runs vet, race, stdlib tests and the WASM suite in a real
  browser. **Tests for the public API must live in `tests/` subdirectory** for better library
  organization — all moveable tests should be relocated there; the goal is to keep the
  majority of tests in that dedicated directory rather than scattered at the root.

---

## Stage 1 — `dom_frontend.go`: rewrite `Show`

Replace the whole current body of `Show` (the re-invocation machinery: `lastSubtreeID`,
per-toggle `cleanupListeners`/`cleanupSignalSubscriptions`/`runCleanups`,
`cleanupChildren`, `innerHTML` re-render) with:

```go
// Show keeps content mounted and toggles its visibility with cond.
// The subtree is built and attached ONCE — a builder re-run that re-attaches
// captured elements (the v0.12 panic) is unrepresentable: there is no builder.
// Hidden means inline display:none on the container, so node identity,
// listeners and signal bindings survive every toggle, and bindings keep
// patching while hidden — the subtree is current the moment it reappears.
func Show(cond *SignalBool, content Component) *Element {
	containerID := generateID()
	container := NewElement("div").ID(containerID)
	if !cond.Get() {
		container.Attr("style", "display:none")
	}
	container.Child(content)

	updater := func() {
		if ref, ok := instance.Get(containerID); ok {
			display := ""
			if !cond.Get() {
				display = "none"
			}
			ref.(*elementWasm).val.Get("style").Set("display", display)
		}
	}
	unsub := cond.subscribe(updater)

	// Register unsub to be called when container is unmounted
	instance.(*domWasm).unsubs = append(instance.(*domWasm).unsubs, struct {
		id    string
		unsub func()
	}{containerID, unsub})

	return container
}
```

Delete nothing else. The container is a normal child: generic unmount
(`cleanupChildren` from the parent side) already tears down its subtree.

### Acceptance

- `grep -n "lastSubtreeID" dom_frontend.go` → empty.
- `grep -n "render func() \*Element" dom_frontend.go dom_backend.go` → empty.

---

## Stage 2 — `dom_backend.go`: same signature for SSR

```go
// Show is implemented for SSR: the child is always serialized; the container
// carries display:none when cond is false, matching the WASM initial markup.
func Show(cond *SignalBool, content Component) *Element {
	container := NewElement("div")
	if !cond.Get() {
		container.Attr("style", "display:none")
	}
	container.Child(content)
	return container
}
```

---

## Stage 3 — tests: convert the reproduction into the permanent guard

### 3a. `show_regression_wasm_test.go` (already on main, red) — rewrite to the new API

Same scenario that panicked in production (content any consumer would naturally share,
two open/close cycles), now the guard. Keep the file name and the `//go:build wasm` tag:

```go
//go:build wasm

package dom

import (
	"testing"
)

// TestShowSecondToggleSharedContent guards the fix for the panic reported in
// tinywasm/layout docs/BUG_DOM.md ("element ... is already a child of another
// element"), which killed the app on the SECOND open of a Show whose render
// callback closed over elements built outside it. Show no longer takes a
// callback: the subtree is built and attached once, so re-attachment is
// unrepresentable. This test pins the consumer scenario — shared content,
// repeated toggles — plus the semantics that come with build-once: node
// identity and live bindings across toggles.
func TestShowSecondToggleSharedContent(t *testing.T) {
	cond := NewBool(false)
	msg := NewString("Delete «laptop»?")
	body := NewElement("div").ID("shared-body").
		Child(NewElement("span").ID("shared-msg").BindText(msg))

	s := Show(cond, body)
	Render("app", s)
	container, ok := Get(s.GetID())
	if !ok {
		t.Fatal("Show container not mounted")
	}
	display := func() string {
		return container.(*elementWasm).val.Get("style").Get("display").String()
	}

	// Hidden at start: mounted, display:none.
	if _, ok := Get("shared-msg"); !ok {
		t.Fatal("content must be mounted while hidden")
	}
	if display() != "none" {
		t.Fatalf("expected display:none while hidden, got %q", display())
	}

	msgRef, _ := Get("shared-msg")
	msgNode := msgRef.(*elementWasm).val

	// Two full open/close cycles — the second open is what panicked before.
	for i := 0; i < 2; i++ {
		cond.Set(true)
		if display() == "none" {
			t.Fatalf("cycle %d: expected visible after Set(true)", i)
		}
		cond.Set(false)
		if display() != "none" {
			t.Fatalf("cycle %d: expected display:none after Set(false)", i)
		}
	}
	cond.Set(true)

	// Node identity survives toggles (no innerHTML re-render).
	if again, _ := Get("shared-msg"); !again.(*elementWasm).val.Equal(msgNode) {
		t.Error("node identity lost across toggles")
	}

	// Bindings keep patching while hidden; the subtree is current on re-show.
	msg.Set("Delete «desktop»?")
	cond.Set(false)
	if got := msgRef.(*elementWasm).val.Get("textContent").String(); got != "Delete «desktop»?" {
		t.Errorf("binding stale while hidden: %q", got)
	}
	cond.Set(true)
	if got := msgRef.(*elementWasm).val.Get("textContent").String(); got != "Delete «desktop»?" {
		t.Errorf("binding stale after re-show: %q", got)
	}
}
```

### 3b. `lifecycle_wasm_test.go` — rewrite `TestShow`

The old version asserts `Get("shown")` FAILS while hidden. Hidden no longer means
unmounted. Replace the whole test:

```go
func TestShow(t *testing.T) {
	cond := NewBool(false)
	s := Show(cond, NewElement("span").ID("shown").Text("visible"))
	Render("app", s)

	// Mounted while hidden — hidden is display:none, not unmounted.
	if _, ok := Get("shown"); !ok {
		t.Fatal("content must stay mounted while hidden")
	}
	container, ok := Get(s.GetID())
	if !ok {
		t.Fatal("Show container not mounted")
	}
	display := func() string {
		return container.(*elementWasm).val.Get("style").Get("display").String()
	}
	if display() != "none" {
		t.Errorf("expected display:none at start, got %q", display())
	}
	cond.Set(true)
	if display() == "none" {
		t.Error("expected visible after Set(true)")
	}
	cond.Set(false)
	if display() != "none" {
		t.Error("expected display:none after Set(false)")
	}
}
```

### 3c. New `show_backend_test.go` (`//go:build !wasm`) — SSR parity

```go
//go:build !wasm

package dom

import (
	"strings"
	"testing"
)

func TestShowBackend(t *testing.T) {
	off := Show(NewBool(false), NewElement("span").Text("x"))
	html := off.String()
	if !strings.Contains(html, "display:none") {
		t.Error("hidden Show must carry display:none in SSR markup")
	}
	if !strings.Contains(html, "<span") {
		t.Error("child must be serialized even while hidden (WASM parity)")
	}

	on := Show(NewBool(true), NewElement("span").Text("x"))
	if strings.Contains(on.String(), "display:none") {
		t.Error("visible Show must not carry display:none")
	}
}
```

### Acceptance

- `gotest` green (vet, race, stdlib, WASM browser suite).
- Public API tests moved to `tests/` subdirectory for consistent library organization.

---

## Stage 4 — docs

### `docs/ARCHITECTURE.md`

Replace the "Reactive Structure" entry (currently line ~155):

```
- `Show(cond *SignalBool, render func() *Element)`: Mounts/unmounts a subtree based on a condition.
```

with:

```
- `Show(cond *SignalBool, content Component)`: A subtree that is always mounted and shown/hidden with
  `display:none` as cond flips. Built and attached ONCE — node identity, listeners and bindings survive
  toggles, and bindings keep patching while hidden.
```

### `README.md`

Re-index so every `docs/` file is linked (add `docs/PLAN.md` if the house convention in
the index allows it; otherwise leave the index untouched — do not invent a new section).

---

## Stages table

| # | File(s) | What lands |
|---|---|---|
| 1 | `dom_frontend.go` | `Show(cond, content)` — build once, `display:none` toggle; re-invocation machinery deleted |
| 2 | `dom_backend.go` | same signature, SSR parity |
| 3 | `show_regression_wasm_test.go`, `lifecycle_wasm_test.go`, `show_backend_test.go` (new) | reproduction converted into the guard; `TestShow` rewritten; SSR test |
| 4 | `docs/ARCHITECTURE.md`, `README.md` | docs |

Stages 1–2 land together (dual build). Stage 3 depends on 1–2. Stage 4 independent.

---

## Definition of done

1. `gotest` green at the module root — vet, race, stdlib and WASM suites.
2. `grep -rn "render func() \*Element" .` → empty (old signature gone everywhere).
3. `grep -rn "lastSubtreeID" .` → empty.
4. `grep -rn "func() \*Element" dom_frontend.go dom_backend.go` → only `DeriveString`/`DeriveBool`-style computes if any; no builder callbacks in `Show`.
5. `GOOS=js GOARCH=wasm go build ./...` compiles.
6. Public API tests organized in `tests/` subdirectory; root-level test files only for internal/integration testing.

## Out of scope

- **Consumers** (`components/modaldialog`, `components/selectsearch`) — different repo,
  `components/docs/PLAN_SHOW.md`, runs after this one is published (gate).
- **`layout` verification** — `layout/docs/PLAN.md`, last stage.
- **Any compatibility shim** (e.g. keeping the old signature under another name). The
  break is the point: the old shape is the trap.
- **A `Hidden` attribute API or CSS-class-based toggling** — inline `display:none` on the
  container is the whole mechanism; alternatives re-open the "author CSS overrides it"
  hole.
