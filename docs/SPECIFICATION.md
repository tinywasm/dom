# TinyDOM Specification

## 1. Philosophy & Goals

*   **Minimalist**: A thin wrapper over the browser DOM, optimized for TinyGo/WASM.
*   **No `syscall/js` exposure**: Components interact *only* with the TinyDOM Go interface.
*   **Zero StdLib**: Uses `tinystring` for string manipulation to keep binary size small.
*   **JSX-like Declarative View**: Uses a type-safe factory pattern (`dom.Div`, etc.) with children varargs for concise UI construction.
*   **Direct & Minimal**: Avoids heavy VDOM tree diffing. Updates are component-level via `c.Update()`, replacing the underlying DOM node directly.
*   **Strongly Typed Form Elements**: Specialized wrappers (`InputEl`, `FormEl`) provide a semantic, type-safe API for form building.
*   **Standard HTML5 Support**: Correct rendering of void elements (`<br>`, `<img>`) and automatic event wiring.
*   **Auto ID Management**: Unique IDs are generated automatically for components and elements with event listeners.
*   **Reactivity**: Components trigger their own re-render via `c.Update()`.

## 2. Architecture

### The "Virtual" DOM (Builder Layer)
The library uses a lightweight element tree structure to:
1.  Define the UI properties (Tags, Attributes, Events) declaratively.
2.  Auto-generate IDs for elements with event listeners.
3.  Hydrate events automatically on the client side.

### Child Component Strategy
TinyDOM automatically manages the lifecycle of child components.
1.  Parent components include child elements or `Component`s in their `Render` method.
2.  The library recursively calls `Render` and `OnMount`.
This ensures consistent initialization and automatic cleanup of event listeners.

 ### Component Contract
A component is a Go struct that:
1.  Embeds `*dom.Element` for identity and lifecycle.
2.  Implements `Render() *dom.Element` or `RenderHTML() string`.
3.  Manages its own state and triggers re-renders via `c.Update()`.

## 3. API Overview

> **Note:** This section provides a high-level overview. For the exact interface definitions and method signatures, please refer to the **[API Reference](API.md)**.

The architecture relies on three core interfaces:

### DOM Interface
The global entry point. It handles:
*   **Lifecycle**: `Render` injects HTML and binds events; `Update` re-renders in place; `Append` adds content.

### Reference Interface
Represents a live DOM node. It provides methods for:
*   **Read**: `GetAttr`, `Value`, `Checked`.
*   **Interaction**: `On` (events), `Focus`.

### Component Interface
The contract for UI parts:
*   `GetID()`: Unique identifier.
*   `SetID(id string)`: Inject unique ID.
*   `RenderHTML()`: Returns static HTML string.
*   `Children()`: Returns child components.

### HTML Builder
The builder functions are part of the `dom` package. They can be dot-imported (`. "github.com/tinywasm/dom"`) for a cleaner DSL if desired.


## 4. Usage Example

> **See [Creating Components](COMPONENTS.md)** for detailed code examples, including basic counters and nested component structures.

## 5. Build Tags Strategy

*   **`dom_wasm.go`**: Implementation using `syscall/js` (hidden).
*   **`dom_stub.go`**: (`!wasm`) No-op implementation for server-side compilation safety.
*   **`css_loader.go`**: (`!wasm`) Logic to extract `RenderCSS()` from components and bundle/serve it.

## 6. Key Design Decisions

1.  **ID Management**: **Automatic**. `dom.Render` and `injectComponentID` handle unique ID generation. Component roots in `Render()` get their ID automatically.
2.  **Child Components**: **Recursive Lifecycle**. The library automatically renders and unmounts child components found in the element tree.
3.  **State Updates**: **Component-Level Reactivity**. Calling `c.Update()` re-renders the component and replaces it in the DOM.
4.  **Builder API**: **Declarative vs Imperative**. We use a Builder pattern (`dom.Div`) to ensure type safety, correct HTML structure, and automated event wiring.
5.  **TinyGo Optimization**: **Slices vs Maps**. To minimize WASM binary size and improve performance, elements use `[]fmt.KeyValue` for attributes and internal slices for events instead of Go maps. Linear scans are faster for typical attribute counts (< 10) and avoid the heavy map runtime support.
6.  **CSS Strategy**: **Global**. Standard CSS classes.
7.  **Routing**: **Single Page Root**. The "Index" is the root component.

## 7. API Design Philosophy (Q&A)

### Why did we remove `Get(id string)`?
*   **Enforce Declarativity**: Imperative DOM manipulation by ID often leads to "spaghetti code" and makes state management unpredictable.
*   **State as Truth**: By removing direct access to nodes, we force the developer to treat the component structure as a projection of state. If you need to change something, update the state and call `Update()`.

### Why `Render(parentID string, ...)`?
*   The target container (e.g., `<div id="app">`) usually exists in the static HTML or is part of a parent's template. It is a stable entry point for the application.
