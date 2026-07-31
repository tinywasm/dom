---
PLAN: "post-mount hook — let a component touch the DOM it just produced"
TAG: v0.12.0
STATUS: running
SESSION: 17029564306673465193
---

# Plan — `dom`: a lifecycle hook that runs *after* the markup is in the document

## The problem, concretely

`Init(ctx Ctx)` is the only lifecycle hook a component has, and it runs **before**
the component's markup exists. In `dom_frontend.go` `Render`:

```go
d.initComponent(component)          // ← Init() fires HERE
...
html = d.renderToHTML(root, &children, component.GetID())
parent.Set("innerHTML", html)       // ← the markup reaches the document HERE
...
d.wireBindings(component.GetID())
for _, child := range children { d.mountRecursive(child) }
```

So any `Get(id)` inside `Init` resolves against a document that does not yet contain
the element. `Get` returns `(nil, false)`, the caller's `if ok` guard skips the work,
and **nothing is reported** — the component initialises "successfully" with an
imperative step silently dropped.

### The consumer that found it

`github.com/tinywasm/layout/platformd` renders its modules as a horizontal deck of
panels and slides between them by scrolling — `display` is a discrete property and
cannot transition, so the movement has to come from the scroller:

```go
// platformd.go — Activate()
p.active.Set(moduleID)
if el, ok := Get(moduleID); ok {
    el.ScrollIntoView()
}
```

`Init` calls `Activate` for the module named in the URL hash. On a cold load at
`#mod1` the result is:

| what | value |
|---|---|
| `data-current` attribute | `"true"` on the `mod1` panel — correct |
| `stage.scrollLeft` | `0` — showing the **crud** panel |

The state is right and the view is wrong. The user sees the wrong module's content
under the right module's nav highlight. Navigating *after* mount works, because the
`hashchange` handler runs when the document is complete — which is what makes this
look intermittent rather than broken.

It is worse than intermittent: on a cold load the browser **also** scrolls to
`#mod1` natively, because the panel carries that id. Sometimes that lands first and
the bug is invisible. The framework is relying on a race it does not control.

### Why this is a `dom` defect and not a `platformd` one

`platformd` cannot fix it. There is no moment it can ask for — the API exposes no
point between "markup inserted" and "user sees it". Per
`app-releases/docs/CONSTRUCTION_HARNESS.md`:

> A missing contract at a boundary is a defect in the library, not in the consumer.

And the failure mode is the one the harness explicitly forbids: not a compile error,
not a loud diagnostic, but a silent no-op.

The workarounds a consumer would otherwise reach for are all worse: polling `Get`
until it succeeds, an arbitrary `setTimeout` (which `dom` does not expose either),
or leaning on the browser's native anchor scroll. Each is a fork of a lifecycle
`dom` should own.

## What to add

One hook, mirroring `Init`, fired after the markup is in the document and the
bindings are wired.

```go
// Mounted runs after the component's markup is in the document and its bindings
// are wired. This is the first moment Get(id) can resolve the component's own
// elements, so it is where imperative DOM work belongs: measuring, scrolling,
// focusing, attaching an observer.
//
// Init is for state. Mounted is for the document. A component that only sets up
// signals never needs this.
type mountable interface {
    Mounted()
}
```

Unexported interface, exported method — the same shape as `initable` already in
`interface.dom.go`, so an author writes `func (c *Thing) Mounted()` and never sees
the interface. That is the existing house pattern and this must not invent a second
one.

### Where it fires

Exactly one place, so there is one answer to "when": at the end of
`mountRecursive`, **after** `wireBindings` and after the component's own children
have mounted.

```go
func (d *domWasm) mountRecursive(c Component) {
	...
	d.wireBindings(c.GetID())

	for _, child := range c.Children() {
		if child != nil {
			d.mountRecursive(child)
		}
	}

	if m, ok := c.(mountable); ok {
		m.Mounted()
	}
}
```

Children first, then the parent. A parent measuring its own subtree needs that
subtree to exist; the reverse is never true.

**The root component is the trap.** `Render` and `Append` call `mountRecursive` on
each *child* but never on the root itself — the root's bindings are wired inline
(`d.wireBindings(component.GetID())`). `platformd` **is** a root: `Append("body", p)`.
If the hook is only wired into `mountRecursive`, the exact consumer this plan exists
for never receives it. The root must be dispatched explicitly at all three entry
points that insert markup: `Render`, `Append`, and `update` (the re-render path).

⚠️ Decide and record whether `Mounted` fires again on re-render (`update`). Both are
defensible; leaving it undecided produces a hook that means different things in
different code paths. The recommendation is **yes, on every insertion**, because
`update` replaces the element wholesale — the old node is gone, and anything a
component attached to it is gone with it. Name the decision in `ARCHITECTURE.md`.

### The backend

`dom_backend.go` renders to a string; there is no document to mount into and
`Get` already returns a stub. `Mounted` must **not** fire there. Nothing to add —
just confirm no shared code path reaches it, and say so in the doc rather than
leaving the next reader to check.

## Fixing the consumer once the hook exists

`platformd` splits `Activate` in two: the state change stays where it is, the scroll
moves to the hook.

```go
func (p *Platform) Mounted() {
    if id := p.active.Get(); id != "" {
        if el, ok := Get(id); ok {
            el.ScrollIntoView()
        }
    }
}
```

That is not part of this plan's deliverable, but the plan is not done until it is
verified against it — see acceptance criterion 4.

## Acceptance criteria

Each is checkable.

1. `gotest` green in `dom`, and `GOOS=js GOARCH=wasm go build ./...` still passes.
2. A test proves the ordering guarantee, not merely that the method is called:
   a component whose `Mounted` does `Get(ownID)` must find its element, and must
   observe its children already mounted. A test that only counts invocations does
   not prove the thing that was broken.
3. A test proves `Mounted` fires for a **root** component passed to `Render` and to
   `Append` — the case `mountRecursive` alone does not cover.
4. **A consumer-shaped test, in `dom`**, per the harness rule that an API is not
   published until one exists: a component that mounts a scrollable strip and
   scrolls to a child from `Mounted`, exercising the real `Get` → `Reference` path.
   If that test is awkward to write, the hook is awkward to use.
5. `docs/ARCHITECTURE.md` gains the lifecycle order — `Init` → render → insert →
   wire bindings → children `Mounted` → own `Mounted` — and the recorded decision
   about re-render.
6. `docs/BINDING_MODEL.md` states plainly that `Get` inside `Init` returns
   `(nil, false)` by construction, so the next person does not rediscover it.
7. `README.md` re-indexes `docs/`.
8. Verified live in `layout/platformd`: a cold load at `#mod1` shows
   `stage.scrollLeft == 375` and the `crudview` title off-screen, on three
   consecutive loads — the current behaviour passes roughly one time in two, so a
   single green run proves nothing.

## Out of scope

- **A general timer/next-tick API** (`SetTimeout`, `RequestAnimationFrame`). `dom`
  exposes neither today. They are a bigger surface with their own cleanup
  semantics, and this hook removes the reason to reach for them here. If a
  component genuinely needs to wait for *paint* rather than for *insertion*, that
  is a separate plan.
- **An unmount counterpart.** `Ctx.OnCleanup` already covers teardown and
  `unmountRecursive` already drives it.
- **Changing `Init`'s timing.** Moving it after insertion would be a breaking change
  for every component that creates its signals there — which is all of them.
- **The duplicate-id guard** added alongside this work (`claimID` in `dom.go`). Same
  family of silent failure, already closed, unrelated to lifecycle.
