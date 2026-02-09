# DOM Refactor - Execution Prompt

**Library**: `github.com/tinywasm/dom`
**Location**: ``
**Status**: Ready to execute
**Estimated Time**: 3-5 days
**Priority**: 🔴 Critical Path (blocks Components and Site)

---

## Context

You are refactoring the `tinywasm/dom` library to implement:
1. **Elm-inspired architecture** (Component-Local State pattern)
2. **Full Fluent Builder API** (chainable methods)
3. **Hybrid rendering** (DSL for dynamic, strings for static)
4. **Auto-ID generation** for all components
5. **Lifecycle hooks** (OnMount, OnUpdate, OnUnmount)

**Goal**: Reduce code verbosity by 30-50%, optimize for TinyGo/WASM (<500KB binaries).

---

## Current State

### Files to Review
Read these files to understand current implementation:
- `dom.go` - Main API
- `component.go` - BaseComponent
- `interface.dom.go` - Interfaces
- `dom_frontend.go` - WASM implementation
- `dom_backend.go` - Backend mock
- `html/builder.go` - DSL implementation
- `web/client.go` - Example usage

### Current Issues
- ❌ Verbose API: Multiple function calls for simple tasks
- ❌ Mixed paradigms: String HTML + DSL coexist without clear pattern
- ❌ Manual re-renders: Must call `dom.Update()` explicitly
- ❌ Incomplete lifecycle: Only `OnMount`/`OnUnmount`, missing `OnUpdate`

### What Already Works (Keep)
- ✅ Auto-ID generation via `generateID()`
- ✅ Build-tag separation (backend vs frontend)
- ✅ Event listener management with cleanup
- ✅ Basic DSL in `dom/html/`

---

## Approved Decisions

All decisions have been made. Implement exactly as specified:

### Decision 1: Elm Architecture Pattern
**Pattern**: Component-Local State with explicit `Update()`

```go
type Counter struct {
    dom.Component
    count int  // Model (state)
}

// View - Declarative rendering
func (c *Counter) Render() dom.Node {
    return dom.Button(
        dom.Text(fmt.Sprint(c.count)),
        dom.OnClick(c.Increment),
    )
}

// Update - State mutation + re-render
func (c *Counter) Increment(e dom.Event) {
    c.count++      // Mutate state
    c.Update()     // Trigger re-render
}
```

**Requirements**:
- Components store state in struct fields (Model)
- `Render()` returns `dom.Node` (View)
- State changes call `Update()` explicitly (Update)
- No message types, no centralized state store

---

### Decision 2: Full Fluent Builder API
**Pattern**: Chainable methods on all DOM elements

```go
dom.Div().
    ID("container").
    Class("flex items-center").
    OnClick(handler).
    Append(
        dom.Button().
            Text("Click me").
            OnClick(clickHandler),
    ).
    Render("body")
```

**Requirements**:
- Every DOM element method returns `*Element` for chaining
- Terminal operations: `Render(parentID)`, `Mount(parentID)`
- Both fluent and functional styles work (backward compatible)
- Optimize for minimal allocations

---

### Decision 3: Hybrid Rendering
**Pattern**: Developer chooses based on component complexity

```go
// Static component → String HTML (smaller binary)
type Header struct {
    dom.BaseComponent
}

func (h *Header) RenderHTML() string {
    return `<header><h1>My App</h1></header>`
}

// Dynamic component → DSL (type-safe)
type Counter struct {
    dom.BaseComponent
    count int
}

func (c *Counter) Render() dom.Node {
    return dom.Div(
        dom.Button(dom.Text("+"), dom.OnClick(c.Increment)),
        dom.Span(dom.Text(fmt.Sprint(c.count))),
    )
}
```

**Requirements**:
- Both `RenderHTML() string` and `Render() dom.Node` are valid
- DOM checks `ViewRenderer` interface first, falls back to `HTMLRenderer`
- No preference enforced, developer decides per component

---

## Implementation Tasks

### Task 1: Update Interfaces (interface.dom.go)

**Add `ViewRenderer` interface**:
```go
// ViewRenderer returns a Node tree for declarative UI
type ViewRenderer interface {
    Render() Node
}
```

**Add lifecycle interfaces** (keep as separate, optional interfaces):
```go
type Mountable interface {
    OnMount()
}

type Updatable interface {
    OnUpdate()
}

type Unmountable interface {
    OnUnmount()
}
```

**Keep existing**:
```go
type Component interface {
    Identifiable
    HTMLRenderer    // RenderHTML() string
    ChildProvider
}
```

---

### Task 2: Implement Fluent Builder (html/builder.go)

**Refactor to return element pointers for chaining**:

```go
type Element struct {
    tag      string
    id       string
    classes  []string
    attrs    map[string]string
    events   []EventHandler
    children []any
}

// Fluent setters (all return *Element)
func (e *Element) ID(id string) *Element {
    e.id = id
    return e
}

func (e *Element) Class(class string) *Element {
    e.classes = append(e.classes, class)
    return e
}

func (e *Element) Attr(key, val string) *Element {
    if e.attrs == nil {
        e.attrs = make(map[string]string)
    }
    e.attrs[key] = val
    return e
}

func (e *Element) OnClick(handler func(Event)) *Element {
    e.events = append(e.events, EventHandler{"click", handler})
    return e
}

func (e *Element) Append(child any) *Element {
    e.children = append(e.children, child)
    return e
}

func (e *Element) Text(text string) *Element {
    e.children = append(e.children, text)
    return e
}

// Terminal operations
func (e *Element) Render(parentID string) error {
    return dom.Render(parentID, e)
}

func (e *Element) Mount(parentID string) error {
    return dom.Render(parentID, e)
}

func (e *Element) ToNode() Node {
    // Convert to Node for use in Render() methods
    return Node{
        Tag: e.tag,
        Attrs: // convert attrs to KeyValue slice
        Events: e.events,
        Children: e.children,
    }
}

// Factory functions
func Div() *Element { return &Element{tag: "div"} }
func Button() *Element { return &Element{tag: "button"} }
func Span() *Element { return &Element{tag: "span"} }
// ... all HTML tags
```

**Backward compatibility**: Keep existing functional helpers:
```go
func Tag(tag string, children ...any) Node { /* existing impl */ }
```

---

### Task 3: Update BaseComponent (component.go)

**Add chainable methods**:
```go
type BaseComponent struct {
    id     string
    prefix string // Optional semantic prefix for debugging
}

func (c *BaseComponent) ID() string {
    if c.id == "" {
        c.id = c.generateID()
    }
    return c.id
}

func (c *BaseComponent) SetID(id string) {
    c.id = id
}

func (c *BaseComponent) generateID() string {
    if c.prefix != "" {
        return c.prefix + "-" + generateID()
    }
    return generateID()
}

// Chainable lifecycle helpers
func (c *BaseComponent) Update() error {
    return Update(c)
}

func (c *BaseComponent) Unmount() {
    Unmount(c)
}

// Default implementations (components override as needed)
func (c *BaseComponent) RenderHTML() string {
    return ""
}

func (c *BaseComponent) Children() []Component {
    return nil
}
```

---

### Task 4: Update Frontend Rendering (dom_frontend.go)

**Update `Render()` to check `ViewRenderer` first**:
```go
func (d *domWasm) Render(parentID string, component Component) error {
    parent, ok := d.Get(parentID)
    if !ok {
        return fmt.Errf("parent element not found: %s", parentID)
    }

    // Generate ID if not set
    if component.ID() == "" {
        component.SetID(generateID())
    }

    // Try ViewRenderer first (DSL), fall back to HTMLRenderer (string)
    var html string
    if vr, ok := component.(ViewRenderer); ok {
        html = d.renderToHTML(vr.Render())
    } else {
        html = component.RenderHTML()
    }

    parent.SetHTML(html)

    // Wire pending events from DSL
    for _, pe := range d.pendingEvents {
        if el, ok := d.Get(pe.id); ok {
            el.On(pe.name, pe.handler)
        }
    }
    d.pendingEvents = nil

    d.mountRecursive(component)
    return nil
}
```

**Add `OnUpdate` support in lifecycle**:
```go
func (d *domWasm) Update(component Component) error {
    id := component.ID()
    el, ok := d.Get(id)
    if !ok {
        return fmt.Errf("component element not found: %s", id)
    }

    // Re-render
    var html string
    if vr, ok := component.(ViewRenderer); ok {
        html = d.renderToHTML(vr.Render())
    } else {
        html = component.RenderHTML()
    }

    elWasm := el.(*elementWasm)
    elWasm.val.Set("outerHTML", html)

    // Clear from cache (element was replaced)
    d.clearCache(id)

    // Wire events
    for _, pe := range d.pendingEvents {
        if el, ok := d.Get(pe.id); ok {
            el.On(pe.name, pe.handler)
        }
    }
    d.pendingEvents = nil

    // Call OnUpdate hook if implemented
    if updatable, ok := component.(Updatable); ok {
        updatable.OnUpdate()
    }

    d.mountRecursive(component)
    return nil
}
```

---

### Task 5: Update Package API (dom.go)

**Add/update public functions**:
```go
// Render injects a component into a parent element (replaces content)
func Render(parentID string, component Component) error {
    if component.ID() == "" {
        component.SetID(generateID())
    }
    return instance.Render(parentID, component)
}

// Append adds a component after last child of parent
func Append(parentID string, component Component) error {
    if component.ID() == "" {
        component.SetID(generateID())
    }
    return instance.Append(parentID, component)
}

// Update re-renders a component in place
func Update(component Component) error {
    return instance.Update(component)
}

// Hydrate attaches event listeners to existing HTML (SSR)
func Hydrate(parentID string, component Component) error {
    return instance.Hydrate(parentID, component)
}

// Unmount removes component and cleans up listeners
func Unmount(component Component) {
    instance.Unmount(component)
}

// Mount is deprecated, use Render
// Kept for backward compatibility
func Mount(parentID string, component Component) error {
    return Render(parentID, component)
}
```

---

### Task 6: Update Example (web/client.go)

**Rewrite to showcase new API**:
```go
//go:build wasm

package main

import (
    "github.com/tinywasm/dom"
    "github.com/tinywasm/fmt"
)

// Example 1: Simple counter with Elm architecture
type Counter struct {
    dom.BaseComponent
    count int
}

func (c *Counter) Render() dom.Node {
    return dom.Div().
        ID(c.ID()).
        Class("counter").
        Append(
            dom.Button().
                Text("-").
                OnClick(c.Decrement),
        ).
        Append(
            dom.Span().
                Class("count").
                Text(fmt.Sprint(c.count)),
        ).
        Append(
            dom.Button().
                Text("+").
                OnClick(c.Increment),
        ).
        ToNode()
}

func (c *Counter) Increment(e dom.Event) {
    c.count++
    c.Update()
}

func (c *Counter) Decrement(e dom.Event) {
    c.count--
    c.Update()
}

func (c *Counter) OnMount() {
    fmt.Println("Counter mounted with ID:", c.ID())
}

// Example 2: Static component with string HTML
type Header struct {
    dom.BaseComponent
}

func (h *Header) RenderHTML() string {
    return `<header class="app-header">
        <h1>DOM Refactor Example</h1>
    </header>`
}

func main() {
    // Render static header
    header := &Header{}
    dom.Render("app", header)

    // Render dynamic counter
    counter := &Counter{count: 0}
    dom.Append("app", counter)

    fmt.Println("App mounted successfully")
    select {}
}
```

---

### Task 7: Update Tests

**Add tests for new functionality**:

`uc_fluent_test.go`:
```go
func TestFluentBuilder(t *testing.T) {
    // Test chainable API
    el := dom.Div().
        ID("test").
        Class("container").
        Append(dom.Button().Text("Click"))

    if el.ID() != "test" {
        t.Error("ID not set")
    }
}
```

`uc_hybrid_render_test.go`:
```go
func TestHybridRendering(t *testing.T) {
    // Test ViewRenderer (DSL)
    type DynamicComp struct {
        dom.BaseComponent
    }
    func (c *DynamicComp) Render() dom.Node {
        return dom.Div()
    }

    // Test HTMLRenderer (string)
    type StaticComp struct {
        dom.BaseComponent
    }
    func (c *StaticComp) RenderHTML() string {
        return "<div>Static</div>"
    }

    // Both should work
}
```

`uc_elm_pattern_test.go`:
```go
func TestElmPattern(t *testing.T) {
    type Counter struct {
        dom.BaseComponent
        count int
    }

    func (c *Counter) Render() dom.Node {
        return dom.Span(dom.Text(fmt.Sprint(c.count)))
    }

    func (c *Counter) Increment() {
        c.count++
        c.Update()
    }

    c := &Counter{}
    // Test state mutation and re-render
}
```

---

## Success Criteria

Before marking as complete, verify:

### ✅ Functionality
- [ ] Fluent builder API works (all methods chainable)
- [ ] Hybrid rendering works (both `Render()` and `RenderHTML()`)
- [ ] Elm pattern works (state + Update + Render)
- [ ] Auto-ID generation for all components
- [ ] Lifecycle hooks: OnMount, OnUpdate, OnUnmount all called correctly
- [ ] Backward compatibility: old functional API still works

### ✅ Testing
- [ ] All existing tests pass
- [ ] New tests added for fluent API
- [ ] New tests added for hybrid rendering
- [ ] New tests added for Elm pattern
- [ ] Run `gotest` (TinyGo compatible)

### ✅ Binary Size
- [ ] Example app compiles with TinyGo
- [ ] WASM binary <500KB (measure with `ls -lh`)
- [ ] No standard library imports in WASM-tagged files

### ✅ Documentation
- [ ] Update `dom/README.md` with new API examples
- [ ] Update `web/client.go` example
- [ ] Add comments to new public APIs

---

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `interface.dom.go` | Modify | Add ViewRenderer, Updatable interfaces |
| `component.go` | Modify | Add chainable methods to BaseComponent |
| `html/builder.go` | Refactor | Implement fluent builder pattern |
| `dom_frontend.go` | Modify | Add ViewRenderer support, OnUpdate hook |
| `dom_backend.go` | Modify | Mirror frontend changes |
| `dom.go` | Modify | Update public API functions |
| `web/client.go` | Rewrite | Showcase new patterns |
| `uc_fluent_test.go` | Create | Test fluent API |
| `uc_hybrid_render_test.go` | Create | Test hybrid rendering |
| `uc_elm_pattern_test.go` | Create | Test Elm pattern |
| `README.md` | Update | Document new API |

---

## Implementation Order

1. **Start with interfaces** (foundation)
2. **Implement fluent builder** (visible API changes)
3. **Update BaseComponent** (scaffolding)
4. **Update frontend/backend rendering** (wiring)
5. **Update public API** (facade)
6. **Rewrite example** (documentation)
7. **Write tests** (validation)
8. **Run gotest** (TinyGo verification)

---

## Questions/Ambiguities?

If you encounter decisions not covered here:
1. **Read** [DOM_API_REDESIGN.md](./DOM_API_REDESIGN.md) for full context
2. **Follow principles**: Minimize code, no magic, TinyGo-first
3. **Prefer**: Explicit over implicit, functional over OOP when in doubt

---

## Completion

When done:
1. Commit changes with message: `refactor(dom): implement Elm pattern, fluent API, and hybrid rendering`
2. Run `gotest` and paste output
3. Measure WASM binary size: `ls -lh dom/web/client.wasm`
4. Report results and move to Phase 2 (Components)

---

**Status**: Ready to execute. All decisions made. Begin implementation.
