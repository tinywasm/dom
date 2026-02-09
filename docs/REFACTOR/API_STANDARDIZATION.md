# TinyWasm Ecosystem - API Standardization Plan

**Status**: ✅ Approved - Ready for Implementation
**Date**: 2026-02-09
**Objective**: Standardize APIs across the TinyWasm ecosystem to minimize code verbosity, maximize developer experience, and optimize binary size for TinyGo/WASM targets.

---

## Executive Summary

This plan refactors five core libraries (`dom`, `components`, `site`, `modules`, `crudp`) to achieve:

1. **30-50% code reduction** - Chainable APIs, smart defaults, auto-generation
2. **Single Responsibility** - Each library has ONE clear purpose
3. **TinyGo-optimized** - <500KB WASM binaries, zero dead code
4. **One clear pattern** - No multiple ways to do the same thing

---

## Core Principles

### 1. Minimize Developer Code (Primary Goal)
- Every API requires minimum tokens to accomplish its task
- Smart defaults over configuration
- Chainable methods over separate function calls
- Auto-generation over manual boilerplate

**Why**: Less code = smaller WASM binary = faster load times.

### 2. TinyGo-First Compatibility
- No standard library dependencies in frontend code
- Use `tinywasm/fmt` instead of `fmt`, `strings`, `strconv`
- Avoid maps where possible (TinyGo map overhead is significant)
- Build-tag separation (`//go:build wasm` vs `//go:build !wasm`)

**Why**: Industry standard for WASM is <500KB. TinyGo achieves this, standard Go does not.

### 3. Single Responsibility Principle

| Library | Single Responsibility |
|---------|----------------------|
| `dom` | Low-level DOM manipulation abstraction over syscall/js |
| `components` | Reusable UI component catalog with SSR/CSR support |
| `site` | Application orchestrator: routing, module lifecycle, asset management |
| `modules` | Business logic modules with storage abstraction (spec only, no implementation) |
| `crudp` | Protocol for front-back communication (deferred) |

### 4. Progressive Disclosure
- Simple things simple (one-liner for basic components)
- Complex things possible (lifecycle hooks when needed)
- Sensible defaults that "just work"

---

## Approved Decisions

All critical decisions have been made. Implementation can proceed with these choices:

### ✅ DOM - Elm Architecture Pattern
**Decision**: **Alternative A - Component-Local State**

```go
type Counter struct {
    dom.Component
    count int  // Model (state)
}

// View
func (c *Counter) Render() dom.Node {
    return Button(Text(c.count), OnClick(c.Increment))
}

// Update
func (c *Counter) Increment(e dom.Event) {
    c.count++
    c.Update() // Explicit re-render
}
```

**Rationale**: Simple, no magic, clear control flow, smallest binary.

---

### ✅ DOM - Chainable API
**Decision**: **Alternative A - Full Fluent Builder**

```go
dom.Div().
    ID("container").
    Class("flex").
    OnClick(handler).
    Append(dom.Button().Text("Click")).
    Render("body")
```

**Rationale**: Very compact, discoverable API, natural chaining.

---

### ✅ DOM - DSL vs String HTML
**Decision**: **Alternative C - Hybrid**

```go
// Static components → String HTML (smaller binary)
func (c *Header) RenderHTML() string {
    return `<h1>Title</h1>`
}

// Dynamic components → DSL (type-safe)
func (c *Counter) Render() dom.Node {
    return Button(...)
}
```

**Rationale**: Best of both worlds - optimize per use case.

---

### ✅ Components - Registration
**Decision**: **Alternative B - Explicit Registry**

```go
site.RegisterHandlers(
    &button.Button{},
    &card.Card{},
    modules.User(),
)
```

**With convenience import**:
```go
import _ "github.com/tinywasm/components/all" // Auto-registers all
```

**Rationale**: Explicit, visible, controllable, tree-shakeable.

---

### ✅ Component vs Module Distinction
**Decision**: **Alternative A - Scope-Based** (renamed from Size-Based)

| Criterion | Component | Module |
|-----------|-----------|--------|
| **Scope** | UI primitive (reusable) | Feature/Page (routable) |
| **Backend** | No backend logic | Has CRUDP operations |
| **URL** | No route | Has route (`#users`) |
| **Location** | `components/` library | `myapp/modules/` user code |

**Decision Rules**:
- ❓ "Has its own URL?" → Yes = Module, No = Component
- ❓ "Talks to backend?" → Yes = Module, No = Component

**Examples**:
- ✅ Component: Button, Card, Input, Table (reusable UI)
- ✅ Module: UserManagement, Dashboard, ProductCatalog (routable pages with backend)

**Rationale**: Clear mental model, avoids ambiguity, organizes code naturally.

---

### ✅ Site - API Simplification
**Decision**: Approved

**Server** (3 lines → 2 lines):
```go
site.RegisterHandlers(modules.Init()...)
site.Serve(":8080") // New one-liner helper
```

**Client** (remove `select {}`):
```go
func main() {
    site.RegisterHandlers(modules.Init()...)
    site.Mount() // Blocks automatically
}
```

---

### ✅ Site - Navigation & Routing
**Decisions**: All approved

1. **Move navigation to `components/nav/`** (navigation IS a UI component)
2. **Add nested routes support** (`#users/123/edit`)
3. **Keep LRU cache = 3 modules** (good balance)
4. **Add optional lifecycle hooks**: `BeforeNavigateAway()`, `AfterNavigateTo()`

---

## Implementation Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                          SITE                                │
│  (Orchestrator: routing, assets, module lifecycle)          │
└───────────────┬──────────────────────────┬──────────────────┘
                │                          │
        ┌───────▼────────┐        ┌────────▼────────┐
        │   COMPONENTS    │        │    MODULES      │
        │  (UI Catalog)   │        │ (Business Logic)│
        └───────┬─────────┘        └────────┬────────┘
                │                           │
        ┌───────▼───────────────────────────▼────────┐
        │               DOM                           │
        │  (Low-level DOM manipulation)               │
        └─────────────────────────────────────────────┘
                          │
                  ┌───────▼────────┐
                  │     CRUDP      │
                  │   (Protocol)    │
                  └─────────────────┘
```

**Data Flow**:
1. **Server**: `modules.Init()` → `site.RegisterHandlers()` → `site.Serve(":8080")`
2. **Client**: `modules.Init()` → `site.RegisterHandlers()` → `site.Mount()`
3. **Lifecycle**: `site` controls navigation → `dom` manages rendering → `components` provide reusable UI

---

## Implementation Sequence

### Phase 1: DOM (3-5 days) - **Critical Path**
See: [DOM_REFACTOR_PROMPT.md](./DOM_REFACTOR_PROMPT.md)

1. Implement Elm-inspired architecture (Component-Local State)
2. Implement Full Fluent Builder API (chainable)
3. Implement Hybrid DSL/String rendering
4. Add auto-ID generation for all components
5. Add lifecycle hooks: `OnMount`, `OnUpdate`, `OnUnmount`
6. Update tests and examples

**Deliverable**: Refactored `dom/` with new API, backward compatible where possible.

---

### Phase 2: Components (2-3 days) - **Depends on DOM**
See: [COMPONENTS_REFACTOR_PROMPT.md](./COMPONENTS_REFACTOR_PROMPT.md)

1. Implement explicit registration system
2. Add convenience import `components/all`
3. Create Phase 1 components: button, card, input, nav, modal, table, form
4. Implement SSR/CSR split with build tags
5. Update `COMPONENT_CREATION.md` guide
6. Write tests for each component

**Deliverable**: Populated `components/` library with 7 essential components.

---

### Phase 3: Site (2-3 days) - **Depends on DOM + Components**
See: [SITE_REFACTOR_PROMPT.md](./SITE_REFACTOR_PROMPT.md)

1. Add `Serve()` helper for server
2. Make client `Mount()` block automatically
3. Implement nested route support (`#users/123/edit`)
4. Move navigation to `components/nav/`
5. Add optional lifecycle hooks (BeforeNavigateAway, AfterNavigateTo)
6. Update examples (server.go, client.go)

**Deliverable**: Refactored `site/` with simplified API.

---

### Phase 4: Documentation (1-2 days)
1. Update all READMEs
2. Create migration guide (old API → new API)
3. Update examples in each library
4. Record video tutorial (optional)

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Code Reduction** | 30-50% less code | LOC in example components before/after |
| **Binary Size** | <500KB | `ls -lh public/client.wasm` |
| **API Clarity** | Developer can create component without docs | User testing |
| **Build Tag Separation** | 0 server code in WASM binary | Audit build artifacts |

---

## Detailed Plans (For Implementation)

Each library has a detailed refactor prompt with step-by-step instructions:

| Library | Prompt Document | Status |
|---------|----------------|--------|
| **DOM** | [DOM_REFACTOR_PROMPT.md](./DOM_REFACTOR_PROMPT.md) | Ready to execute |
| **Components** | [COMPONENTS_REFACTOR_PROMPT.md](./COMPONENTS_REFACTOR_PROMPT.md) | Ready to execute |
| **Site** | [SITE_REFACTOR_PROMPT.md](./SITE_REFACTOR_PROMPT.md) | Ready to execute |
| **Modules** | [MODULES_SPECIFICATION.md](./MODULES_SPECIFICATION.md) | Documentation only (no implementation) |

---

## Parallel Execution Strategy

**These phases can be partially parallelized**:

1. **Phase 1 (DOM)** must complete first (critical path)
2. **Phase 2 (Components)** and **Phase 3 (Site)** can start in parallel once DOM API is locked
3. Different agents/developers can work on different phases simultaneously

**Workflow**:
```
Agent 1: DOM refactor (days 1-5)
         ↓ (API locked)
         ↓
Agent 2: ├─→ Components (days 6-8)
Agent 3: └─→ Site (days 6-8)
         ↓
Agent 4: Documentation (days 9-10)
```

---

## Testing Strategy

### Unit Tests (No Browser)
Test components in isolation:
```go
func TestButton_Render(t *testing.T) {
    btn := &Button{Text: "Click"}
    node := btn.Render()
    // Assert node structure
}
```

### Integration Tests (WASM)
Use `dom_backend.go` mock or E2E tests:
```go
func TestCounter_Integration(t *testing.T) {
    c := &Counter{}
    dom.Render("body", c)
    // Verify rendering and lifecycle
}
```

### Performance Tests
Measure binary size and load time:
```bash
$ wasm-opt -O3 public/client.wasm -o optimized.wasm
$ ls -lh optimized.wasm
```

---

## Migration Guide (Backward Compatibility)

### Breaking Changes
1. `dom.Mount()` → `dom.Render()` (kept as deprecated alias)
2. Component interface unchanged (fully backward compatible)
3. Fluent Builder is **additive** (old functional style still works)

### Non-Breaking Additions
1. Chainable methods on `BaseComponent`
2. `ViewRenderer` interface (optional, coexists with `HTMLRenderer`)
3. `site.Serve()` helper (additive, doesn't remove `Mount()`)

### Migration Steps
1. **Optional**: Adopt fluent builder syntax where it reduces code
2. **Optional**: Use DSL for dynamic components, strings for static
3. **Required**: Update imports if moving from old package locations

**Timeline**: Gradual migration over 3-6 months, deprecate old patterns in v2.0.

---

## Next Steps

1. ✅ Review and approve this plan (DONE)
2. 🔄 Execute refactor prompts in sequence:
   - [DOM_REFACTOR_PROMPT.md](./DOM_REFACTOR_PROMPT.md)
   - [COMPONENTS_REFACTOR_PROMPT.md](./COMPONENTS_REFACTOR_PROMPT.md)
   - [SITE_REFACTOR_PROMPT.md](./SITE_REFACTOR_PROMPT.md)
3. 🔄 Run `gotest` after each phase to ensure TinyGo compatibility
4. 🔄 Update documentation and examples
5. 🔄 Publish new versions with migration guide

---

## Questions During Implementation?

Refer back to detailed plans for context:
- [DOM_API_REDESIGN.md](./DOM_API_REDESIGN.md) - Full architecture rationale
- [COMPONENTS_STRUCTURE.md](./COMPONENTS_STRUCTURE.md) - Component patterns
- [SITE_ORCHESTRATION.md](./SITE_ORCHESTRATION.md) - Module lifecycle

If ambiguity arises, follow these principles:
1. **Minimize code** - Choose the option that requires fewer lines
2. **No magic** - Explicit over implicit
3. **TinyGo-first** - Avoid features that bloat binaries
4. **One way** - Don't add multiple APIs for the same task

---

**Status**: ✅ Ready for implementation. All decisions finalized.
