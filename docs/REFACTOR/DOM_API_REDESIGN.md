# DOM - API Redesign Plan

**Parent Plan**: [API_STANDARDIZATION.md](./API_STANDARDIZATION.md)
**Library**: `github.com/tinywasm/dom`
**Status**: Draft - Awaiting Critical Decisions
**Priority**: 🔴 Critical Path (Blocks Components & Site)

---

## Current State Analysis

### What Works ✅
- Auto-ID generation via `generateID()`
- Clean separation: backend (`dom_backend.go`) vs frontend (`dom_frontend.go`)
- Event listener management with automatic cleanup
- Basic lifecycle: `OnMount` / `OnUnmount` via `Mountable` interface
- HTML DSL partially implemented (`dom/html/builder.go`)

### What's Problematic ❌
- **Verbose API**: Requires multiple function calls for simple tasks
  ```go
  dom.Render("body", c)
  el, _ := dom.Get("btn")
  el.On("click", handler)
  dom.Update(c)
  ```
- **Mixed paradigms**: String HTML (`RenderHTML()`) + DSL (`Render()`) + imperative (`Get().SetHTML()`)
- **Unclear lifecycle**: When is `OnMount` called? What about `OnUpdate`?
- **No state management pattern**: Developers freestyle state updates
- **Manual re-renders**: Must call `dom.Update(c)` explicitly after state changes

### Uncommitted Changes (Mixed State)
Git diff shows:
- ✅ Good: `Render/Append/Hydrate/Update` methods (semantic naming)
- ✅ Good: DSL support via `ViewRenderer` interface
- ✅ Good: Pending events system for DSL
- ❌ Mixed: Both `RenderHTML()` string and `Render()` Node coexist (choose one)

**Decision**: Keep the good parts, remove redundancy, add missing pieces.

---

## Single Responsibility

**DOM's ONLY job**: Provide a minimal, type-safe abstraction over `syscall/js` for DOM manipulation.

**NOT DOM's job**:
- Business logic (that's `modules`)
- Reusable UI patterns (that's `components`)
- Routing/Navigation (that's `site`)
- Data fetching (that's `crudp`)

**Boundary**: DOM stops at "render tree to browser" and "listen to events". Everything else is higher-level.

---

## Elm-Inspired Architecture (Adapted for Go)

### Classic Elm
```elm
type Msg = Increment | Decrement

update : Msg -> Model -> Model
update msg model =
    case msg of
        Increment -> { model | count = model.count + 1 }
        Decrement -> { model | count = model.count - 1 }

view : Model -> Html Msg
view model =
    div []
        [ button [ onClick Increment ] [ text "+" ]
        , text (String.fromInt model.count)
        ]
```

### Go/WASM Adaptation (3 Alternatives)

#### ⭐ **Alternative A: Component-Local State (RECOMMENDED)**

```go
type Counter struct {
    dom.Component // Embeds: ID, lifecycle
    count int      // Model (state)
}

// View: Declarative UI from current state
func (c *Counter) View() dom.Node {
    return dom.Div(
        dom.Button(dom.Text("+"), dom.OnClick(c.Increment)),
        dom.Span(dom.Text(fmt.Sprint(c.count))),
    )
}

// Update: State mutation + re-render
func (c *Counter) Increment(e dom.Event) {
    c.count++ // Mutate model
    c.Render() // Trigger re-render (uses View())
}

// Usage
c := &Counter{count: 0}
c.Mount("body") // Chainable: auto-generates ID, renders, calls OnMount
```

**Pros**:
- ✅ Minimal code (no msg types, no switch statements)
- ✅ Clear ownership: component owns its state
- ✅ Explicit re-renders: no magic, easy to debug
- ✅ Go-idiomatic: methods on structs, mutation is normal
- ✅ Small binary: no generics, no reflection, no reactive runtime

**Cons**:
- ❌ Manual `Render()` call after state changes (but can macro this later)
- ❌ Testing requires instantiating components (but that's fine)

**Justification**: Best balance of "less code" and "no magic" for Go developers. Aligns with Go's philosophy of explicitness.

---

#### **Alternative B: Message-Based (Pure Elm)**

```go
type CounterMsg int
const (
    Increment CounterMsg = iota
    Decrement
)

type Counter struct {
    dom.Component
    count int
}

// Update: Pure function (testable)
func (c *Counter) Update(msg CounterMsg) {
    switch msg {
    case Increment: c.count++
    case Decrement: c.count--
    }
    c.Render() // Auto-called by dispatcher
}

// View: Returns virtual DOM
func (c *Counter) View() dom.Node {
    return dom.Button(
        dom.Text(fmt.Sprint(c.count)),
        dom.OnClick(func() { c.Dispatch(Increment) }),
    )
}
```

**Pros**:
- ✅ Pure `Update()` function (testable without DOM)
- ✅ Centralized state transitions (easy to log/replay)
- ✅ Type-safe messages (compile-time guarantees)

**Cons**:
- ❌ More boilerplate: define msg type, write switch statement
- ❌ Indirection: click → dispatch → update → render (harder to trace)
- ❌ Larger binaries: more types, more code paths

**Justification**: Too verbose for Go. Works in Elm because of language features (pattern matching, compiler optimizations) we don't have.

---

#### **Alternative C: Reactive State (Auto-Update)**

```go
type Counter struct {
    dom.Component
    count *dom.Signal[int] // Reactive primitive
}

func NewCounter() *Counter {
    c := &Counter{}
    c.count = dom.NewSignal(0)
    c.count.OnChange(func() { c.Render() }) // Auto re-render
    return c
}

func (c *Counter) View() dom.Node {
    return dom.Button(
        dom.Text(fmt.Sprint(c.count.Get())),
        dom.OnClick(func() { c.count.Set(c.count.Get() + 1) }),
    )
}
```

**Pros**:
- ✅ Minimal code: no manual `Render()` calls
- ✅ Feels modern (React-like)

**Cons**:
- ❌ Magic behavior: mutations trigger renders (hard to debug)
- ❌ Larger runtime: Signal tracking, dependency graphs
- ❌ Not TinyGo-friendly: likely uses generics, reflection
- ❌ Surprising: changing `count.Set()` has side effects

**Justification**: Too much magic, too large for WASM. Defeats the "TinyGo-first" principle.

---

### 🎯 Recommended Decision

**Use Alternative A (Component-Local State)** because:
1. Smallest binary size (no framework overhead)
2. Most Go-idiomatic (methods, mutation, explicit calls)
3. Easiest to debug (no indirection, no magic)
4. Satisfies "less code" (no msg types, no boilerplate)

**Trade-off**: Developers must call `.Render()` after state changes. This is acceptable because:
- It's explicit and predictable
- Can be optimized later with code generation if needed
- Most components have 1-2 state-mutating methods (not a huge burden)

---

## Chainable API Design

### Current (Functional Style)
```go
c := &Counter{}
dom.Render("body", c)
el, ok := dom.Get("btn")
if ok {
    el.On("click", handler)
}
dom.Update(c)
```

**Problem**: Requires 4+ lines for common operations, separate function calls, error handling.

### Proposed (Hybrid Chainable)

#### ⭐ **Option 1: Component-Level Chaining (RECOMMENDED)**

```go
// Component embeds dom.Component which provides:
type Component struct {
    id string
}

func (c *Component) Mount(parentID string) *Component {
    dom.Render(parentID, c)
    return c
}

func (c *Component) Render() *Component {
    dom.Update(c)
    return c
}

// Usage
c := &Counter{}
c.Mount("body").OnReady(func() {
    fmt.Println("Component mounted")
})
```

**Pros**:
- ✅ Compact: `c.Mount("body")` vs `dom.Render("body", c)`
- ✅ Discoverable: IDE autocomplete shows available methods
- ✅ Chainable: `c.Mount("body").OnMount(fn)`
- ✅ Backward compatible: `dom.Render()` still works

**Cons**:
- ❌ Slightly larger API surface (methods on `Component` + package functions)

---

#### **Option 2: Full Fluent Builder**

```go
dom.New("div").
    ID("container").
    Class("flex").
    Children(
        dom.New("button").Text("Click").OnClick(handler),
    ).
    Mount("body")
```

**Pros**:
- ✅ Very compact, fully chainable
- ✅ No need to define component structs for simple UI

**Cons**:
- ❌ Different return types at each step (complex to implement)
- ❌ Harder to reuse: can't save intermediate state
- ❌ Doesn't fit Elm model (no clear separation of Model/View/Update)

---

#### **Option 3: Keep Current (Functional)**

```go
// No chaining, just functions
dom.Render("body", c)
dom.Update(c)
```

**Pros**:
- ✅ Simple to implement and understand

**Cons**:
- ❌ Verbose, doesn't reduce code

---

### 🎯 Recommended Decision

**Use Option 1 (Component-Level Chaining)** because:
- Natural fit for component model
- Reduces code without adding complexity
- Easy to implement (`return c` at end of methods)
- Backward compatible with functional style

---

## DSL vs String HTML

### Current State
Two rendering paths exist:
1. **String HTML**: `RenderHTML() string`
2. **DSL Nodes**: `Render() dom.Node`

**Problem**: Redundancy. Which should developers use?

### Alternatives

#### ⭐ **Option A: Hybrid Based on Complexity (RECOMMENDED)**

**Rule**:
- Static components → Use string HTML (smaller binary)
- Dynamic/Interactive components → Use DSL (type safety, composability)

```go
// Static: Pure HTML string
type Header struct { dom.Component }
func (h *Header) RenderHTML() string {
    return `<header><h1>Site Title</h1></header>`
}

// Dynamic: DSL for type safety
type Counter struct {
    dom.Component
    count int
}
func (c *Counter) Render() dom.Node {
    return dom.Div(
        dom.Button(dom.Text("+"), dom.OnClick(c.Increment)),
        dom.Span(dom.Text(fmt.Sprint(c.count))),
    )
}
```

**Detection**: DOM checks `ViewRenderer` interface first, falls back to `HTMLRenderer`:
```go
if vr, ok := c.(ViewRenderer); ok {
    return renderNode(vr.Render()) // DSL path
}
return c.RenderHTML() // String path
```

**Pros**:
- ✅ Best of both worlds: size optimization + type safety
- ✅ Developer chooses based on use case
- ✅ No migration pain: both work

**Cons**:
- ❌ Two patterns to learn (but clear rule: static=string, dynamic=DSL)

---

#### **Option B: DSL Only**

Force all components to use DSL:
```go
func (c *Component) Render() dom.Node { ... }
```

**Pros**:
- ✅ One way to do things
- ✅ Type-safe, refactorable

**Cons**:
- ❌ Larger binaries for simple static components
- ❌ Forces DSL on developers who prefer HTML strings

---

#### **Option C: Strings Only**

Remove DSL entirely, use only `RenderHTML()`:
```go
func (c *Counter) RenderHTML() string {
    return fmt.Html(`<button>%d</button>`, c.count)
}
```

**Pros**:
- ✅ Smallest binary size
- ✅ Familiar to web developers

**Cons**:
- ❌ No type safety (typos in HTML tags)
- ❌ No composability (can't nest `dom.Node` objects)
- ❌ Event handlers require manual ID management

---

### 🎯 Recommended Decision

**Use Option A (Hybrid)** because:
- Developers can optimize per use case
- Clear rule: "Is it interactive? Use DSL. Is it static? Use string."
- Doesn't force one paradigm on all scenarios
- Smaller binaries for static content, type safety for dynamic

---

## Lifecycle Hooks

### Current State
- `OnMount()` exists via `Mountable` interface
- `OnUnmount()` exists for cleanup
- No `OnUpdate()` or `AfterRender()`

### Proposed Enhancements

#### Full Lifecycle Hooks

```go
type Lifecycle interface {
    BeforeMount() // Called before HTML injection (rare)
    OnMount()     // Called after HTML is in DOM (common)
    OnUpdate()    // Called after re-render (common)
    OnUnmount()   // Called before removal (common)
}
```

**Problem**: Forces every component to implement 4 methods (even if empty).

**Solution**: Make them optional via separate interfaces:

```go
// Core (always called)
type Mountable interface {
    Component
    OnMount()
}

// Optional (only if implemented)
type Updatable interface {
    OnUpdate()
}

type Unmountable interface {
    OnUnmount()
}

type BeforeMounter interface {
    BeforeMount()
}
```

**DOM's responsibility**: Check for these interfaces and call them if present.

```go
func (d *domWasm) Render(parentID string, c Component) error {
    if bm, ok := c.(BeforeMounter); ok {
        bm.BeforeMount()
    }

    // Inject HTML...

    if m, ok := c.(Mountable); ok {
        m.OnMount()
    }
    return nil
}
```

**Pros**:
- ✅ Components only implement what they need
- ✅ No empty stub methods
- ✅ Backward compatible

---

## Auto-ID Generation

### Current State
IDs are auto-generated via `generateID()` which returns sequential integers: `"1"`, `"2"`, `"3"`.

### Enhancement: Semantic Prefixes (Optional)

Allow components to provide a prefix for debugging:

```go
type Component struct {
    id     string
    prefix string // Optional
}

func (c *Component) ID() string {
    if c.id == "" {
        if c.prefix != "" {
            c.id = c.prefix + "-" + generateID()
        } else {
            c.id = generateID()
        }
    }
    return c.id
}
```

**Usage**:
```go
c := &Counter{prefix: "counter"}
c.Mount("body") // ID will be "counter-1", "counter-2", etc.
```

**Pros**:
- ✅ Easier to debug (HTML inspector shows `<div id="counter-1">`)
- ✅ Optional (default is just numbers)
- ✅ Small overhead (1 string concatenation)

**Cons**:
- ❌ Slightly larger binary (string concat code)

**Decision**: Include it. The debugging benefit outweighs the tiny cost.

---

## API Summary (Final)

### Core Component Interface

```go
package dom

// Component is the minimal interface all components must implement
type Component interface {
    ID() string
    SetID(string)
    RenderHTML() string // Fallback (string-based rendering)
}

// ViewRenderer is optional for DSL-based rendering
type ViewRenderer interface {
    Component
    Render() Node // Returns declarative UI tree
}

// Lifecycle hooks (all optional)
type Mountable interface {
    OnMount() // After HTML injected
}

type Updatable interface {
    OnUpdate() // After re-render
}

type Unmountable interface {
    OnUnmount() // Before removal
}

// BaseComponent provides default implementations
type BaseComponent struct {
    id     string
    prefix string
}

func (c *BaseComponent) ID() string { ... }
func (c *BaseComponent) SetID(id string) { ... }
func (c *BaseComponent) RenderHTML() string { return "" }

// Chainable methods
func (c *BaseComponent) Mount(parentID string) *BaseComponent {
    Render(parentID, c)
    return c
}

func (c *BaseComponent) Render() *BaseComponent {
    Update(c)
    return c
}
```

### Package-Level Functions

```go
// Primary API (backward compatible)
func Render(parentID string, c Component) error
func Append(parentID string, c Component) error
func Hydrate(parentID string, c Component) error
func Update(c Component) error
func Unmount(c Component)

// Element access
func Get(id string) (Element, bool)
func QueryAll(selector string) []Element

// Routing helpers
func OnHashChange(handler func(hash string))
func GetHash() string
func SetHash(hash string)
```

### Example Component (Elm-Style)

```go
package main

import "github.com/tinywasm/dom"

// Model (state)
type Counter struct {
    dom.BaseComponent
    count int
}

// View (rendering)
func (c *Counter) Render() dom.Node {
    return dom.Div(
        dom.Button(
            dom.Text("+"),
            dom.OnClick(c.Increment),
        ),
        dom.Span(dom.Text(fmt.Sprint(c.count))),
        dom.Button(
            dom.Text("-"),
            dom.OnClick(c.Decrement),
        ),
    )
}

// Update (state mutations)
func (c *Counter) Increment(e dom.Event) {
    c.count++
    c.Render() // Re-render with new state
}

func (c *Counter) Decrement(e dom.Event) {
    c.count--
    c.Render()
}

// Lifecycle (optional)
func (c *Counter) OnMount() {
    fmt.Println("Counter mounted with ID:", c.ID())
}

func main() {
    c := &Counter{count: 0}
    c.Mount("body") // Chainable, auto-generates ID
    select {}
}
```

**Lines of code**: ~30 (vs ~50 in current API)
**Binary size**: Estimated ~30KB for this component (TinyGo optimized)

---

## Migration Path

### Phase 1: Add (Non-Breaking)
1. Add `ViewRenderer` interface
2. Add chainable methods to `BaseComponent`
3. Add `OnUpdate` hook detection
4. Add semantic ID prefixes

### Phase 2: Deprecate (Warning)
1. Mark `Mount()` as deprecated in favor of `Render()`
2. Add deprecation notices in docs

### Phase 3: Remove (Breaking - v2.0)
1. Remove deprecated functions
2. Remove `RenderHTML()` fallback (DSL only) - **OPTIONAL**

**Timeline**: Phase 1 can ship immediately. Phase 2 after 3 months. Phase 3 is optional.

---

## Open Questions for Approval

### ❓ Q1: Elm Pattern
**Approve Alternative A (Component-Local State)?**
- [ ] Yes, proceed with Alt A
- [ ] No, use Alt B (Message-Based)
- [ ] No, use Alt C (Reactive)
- [ ] Need more examples to decide

### ❓ Q2: Chaining
**Approve Option 1 (Component-Level Chaining)?**
- [ ] Yes, proceed with Option 1
- [ ] No, use Option 2 (Full Fluent Builder)
- [ ] No, keep current functional style

### ❓ Q3: DSL Strategy
**Approve Option A (Hybrid: String for static, DSL for dynamic)?**
- [ ] Yes, allow both rendering styles
- [ ] No, force DSL only (Option B)
- [ ] No, force strings only (Option C)

### ❓ Q4: Semantic ID Prefixes
**Include optional semantic prefixes for debugging?**
- [ ] Yes, include it
- [ ] No, keep numeric IDs only

---

## Next Steps After Approval

1. Implement `ViewRenderer` interface
2. Add chainable methods to `BaseComponent`
3. Update `dom_frontend.go` to detect `ViewRenderer` vs `HTMLRenderer`
4. Add `OnUpdate` hook detection
5. Update examples and docs
6. Run `gotest` to ensure TinyGo compatibility
7. Measure binary size impact

**Estimated Effort**: 2-3 days implementation + 1 day testing/docs

---

**Ready for approval?** Once you answer the questions above, I'll finalize the [COMPONENTS_STRUCTURE.md](./COMPONENTS_STRUCTURE.md) plan.
