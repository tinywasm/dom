# Comparison & Trade-offs

This document analyzes how TinyDOM compares to other approaches for building web interfaces in Go (WASM).

## 1. TinyDOM vs. Raw `syscall/js`

Using the standard library's `syscall/js` directly is the most low-level approach.

| Feature | TinyDOM | Raw `syscall/js` |
| :--- | :--- | :--- |
| **Abstraction** | High. Interfaces (`DOM`, `Element`) hide JS details. | None. Direct JS value manipulation. |
| **Safety** | High. Type-safe methods. | Low. `js.Value` is `interface{}`-like; prone to panics. |
| **Memory** | **Automatic**. Tracks and releases event listeners on `Unmount`. | **Manual**. Easy to leak memory (forgetting `Release()`). |
| **Performance** | **Cached**. IDs are mapped to JS objects to avoid lookups. | **Slow**. Repeated `Call("getElementById")` is expensive. |
| **Binary Size** | Small overhead (interfaces + cache map). | Smallest possible. |

**Verdict**: TinyDOM adds essential safety and caching with negligible binary overhead, making it far superior for application development.

## 2. TinyDOM vs. VDOM Libraries (Vecty, Go-App)

Libraries like Vecty or Go-App implement a full Virtual DOM (React-style) with diffing algorithms.

| Feature | TinyDOM | VDOM Libraries |
| :--- | :--- | :--- |
| **Update Strategy** | **Direct**. You call `SetText`. Exact & fast. | **Declarative**. You change state, lib diffs tree. |
| **Performance** | **Predictable**. No diffing overhead. | **Variable**. Diffing large trees in WASM can be slow. |
| **Binary Size** | **Tiny**. Minimal logic. | **Large**. Diffing engine + reconciliation logic. |
| **Developer Exp.** | **Imperative**. "On click, update text". | **Declarative**. "State is X, view is Y". |
| **Focus** | **TinyGo**. Optimized for small binaries. | **Standard Go**. Often too heavy for TinyGo. |

**Verdict**: Use **TinyDOM** if binary size and raw performance are critical, or if you prefer simple control flow. Use **VDOM** if you have complex, deeply nested state and don't mind larger binaries (10MB+ vs 500KB).

## 3. TinyDOM vs. JavaScript Frameworks (React, Vue)

| Feature | TinyDOM (Go/WASM) | JS Frameworks |
| :--- | :--- | :--- |
| **Language** | Go. Strong typing, shared backend logic. | JavaScript/TypeScript. |
| **Ecosystem** | Small (Go WASM is niche). | Massive (NPM). |
| **Load Time** | Slower (WASM download + compile). | Faster (JS parses quickly). |
| **Runtime** | Near-native speed for logic. | V8 Engine speed. |

**Verdict**: Use **TinyDOM** if you want to write Go, share domain logic with your backend, or target embedded/IoT devices with web UIs. Use **JS** if you need extensive existing UI libraries (Datepickers, Charts) immediately.

## Summary of Trade-offs

### ✅ Pros of TinyDOM
*   **Tiny Binaries**: Designed specifically for TinyGo.
*   **No Magic**: You know exactly when and how the DOM updates.
*   **Memory Safety**: Solves the hardest part of Go WASM (event listener leaks).
*   **Backend Compatible**: Interfaces allow mocking for server-side mounting or testing.

### ❌ Cons of TinyDOM
*   **Manual Updates**: You must explicitly tell the DOM what to change (`SetText`). No "reactive" magic.
*   **Boilerplate**: Requires defining IDs and struct methods for components.
*   **No Component Library**: You build your own buttons, inputs, etc. (HTML/CSS).
