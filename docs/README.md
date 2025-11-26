# TinyDOM Documentation

Welcome to the TinyDOM documentation. This library provides a minimalist, WASM-optimized way to interact with the browser DOM in Go, avoiding the overhead of the standard library and `syscall/js` exposure.

## 📚 Core Documentation

1.  **[Specification & Philosophy](SPECIFICATION.md)**
    *   Start here to understand the design goals: Minimalist, ID-based caching, and direct DOM manipulation.
    *   Explains the architecture and key decisions (Manual IDs, Manual Cascade).

2.  **[API Reference](API.md)**
    *   Detailed definition of the core interfaces:
        *   `DOM`: The main entry point (`Get`, `Mount`, `Unmount`).
        *   `Element`: Node manipulation (`SetText`, `AddClass`, `Click`).
        *   `Component`: The contract for your UI parts (`RenderHTML`, `OnMount`).

3.  **[Creating Components](COMPONENTS.md)**
    *   Practical guide to building components.
    *   How to create basic components.
    *   How to handle nested components (Manual Cascade pattern).
    *   CSS handling strategy.

4.  **[Advanced Patterns](ADVANCED.md)**
    *   **Dynamic Lists**: How to efficiently add/remove items using `AppendHTML` and `Unmount` without re-rendering lists.
    *   **Decoupling**: Using narrow interfaces (`TextSetter`) to keep components testable and loosely coupled.

5.  **[Comparison & Trade-offs](COMPARISON.md)**
    *   Why TinyDOM?
    *   vs. Raw `syscall/js` (Safety & Caching).
    *   vs. VDOM Libraries like Vecty (Binary Size & Performance).
    *   vs. JS Frameworks (Language & Ecosystem).

## 🚀 Implementation Roadmap

If you are contributing or implementing the core, follow this order:

1.  **Interfaces**: Define `dom.go` and `element.go` based on [API.md](API.md).
2.  **Stubs**: Create `!wasm` implementations for backend compatibility.
3.  **WASM Core**: Implement `dom_wasm.go` (Cache & Lifecycle).
4.  **Events**: Implement the event listener tracking and cleanup system.
