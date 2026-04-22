# PLAN: dom — Component embedding rule

## Problem

Components that embed `*dom.Element` (pointer) and implement `ViewRenderer` panic at runtime
when passed as `dom.Component` slots (e.g. to `rightpanel.RightPanel`).

**Root cause:** `dom.(*domWasm).renderToHTML` calls `v.GetID()` on any `Component` child before
calling `Render()`. If the embedded `*dom.Element` is a nil pointer, `GetID()` dereferences nil → panic.

Stack trace observed:
```
panic: runtime error: invalid memory address or nil pointer dereference
github.com/tinywasm/dom.(*Element).GetID(...)
    element.go:76
github.com/tinywasm/dom.(*domWasm).renderToHTML(...)
    dom_frontend.go:343
```

## Rule

**All consumer structs that implement `dom.Component` or `dom.ViewRenderer` must embed
`dom.Element` as a value, not as a pointer.**

```go
// ❌ Wrong — nil pointer panic when passed as dom.Component slot
type MyView struct {
    *dom.Element  // pointer — nil by default
    ...
}

// ✅ Correct — zero value Element, never nil
type MyView struct {
    dom.Element  // value — always initialized
    ...
}
```

## Why this happens

`renderToHTML` (dom_frontend.go:343) calls `v.GetID()` on every `Component` child to ensure
it has an ID before rendering. This happens **before** checking `ViewRenderer`, so even views
that implement `Render() *dom.Element` must have a valid (non-nil) embedded Element.

## Action items

- [x] Add a guard in `renderToHTML` to check for nil before calling `GetID()` (defensive fix)
  - Added check `if v != nil` at dom_frontend.go:343 before calling `v.GetID()`
- [x] Add a note in `interface.dom.go` documenting the value-embed requirement
  - Added NOTE block in Component docstring with correct and incorrect examples
- [ ] Consider providing a `BaseComponent` zero-value struct for consumers to embed
  - Deferred: may add in future if needed; value-embed pattern is sufficient for now
