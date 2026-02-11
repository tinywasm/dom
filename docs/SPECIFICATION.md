# TinyDOM Specification

## 1. Philosophy & Goals

*   **Minimalist**: A thin wrapper over the browser DOM, optimized for TinyGo/WASM.
*   **No `syscall/js` exposure**: Components interact *only* with the TinyDOM Go interface.
*   **Zero StdLib**: Uses `tinystring` for string manipulation to keep binary size small.
*   **Declarative Builder**: Uses a type-safe `dom.Node` builder pattern (`dom.Div`, etc.) for UI construction.
*   **Direct & Cached**: Instead of a heavy VDOM tree diffing algorithm, it uses an **ID-based caching mechanism** and direct DOM updates via `c.Update()`.
*   **Standard HTML/CSS**: Components can still use raw HTML strings if needed, but the Builder API is preferred.
*   **Dependency Injection**: The `DOM` is injected into components, not imported globally.
*   **Auto ID Management**: The library automatically assigns unique IDs to components and event targets.
*   **Reactivity**: Components trigger their own re-render via `dom.Update(c)`.

## 2. Architecture

### The "Virtual" DOM (Builder Layer)
The library uses a lightweight `Node` tree structure to:
1.  Define the UI properties (Tags, Attributes, Events) declaratively.
2.  Auto-generate IDs for elements with event listeners.
3.  Hydrate events automatically on the client side.

### Child Component Strategy
TinyDOM automatically manages the lifecycle of child components.
1.  Parent components include child `Node`s or `Component`s in their `Render` method.
2.  The library recursively calls `Render` and `OnMount`.
This ensures consistent initialization and automatic cleanup of event listeners.

### Component Contract
A component is a Go struct that:
1.  Implements the `Identifiable` interface (`GetID()` and `SetID()`).
2.  Implements methods to return its HTML/CSS.
3.  Manages its own events via the global API.

## 3. API Overview

> **Note:** This section provides a high-level overview. For the exact interface definitions and method signatures, please refer to the **[API Reference](API.md)**.

The architecture relies on three core interfaces:

### DOM Interface
The entry point injected into components. It handles:
*   **Caching**: `Get(id)` returns a cached `Element`.
*   **Lifecycle**: `Render` injects HTML and binds events; `Unmount` cleans them up; `Update` re-renders in place.

### Element Interface
Represents a DOM node. It provides methods for:
*   **Content**: `SetText`, `SetHTML`, `AppendHTML`.
*   **Attributes**: `AddClass`, `SetAttr`, `Value`.
*   **Events**: `Click`, `On` (with automatic cleanup).
*   **Manipulation**: `Remove`.

### Component Interface
The contract for UI parts:
*   `GetID()`: Unique identifier.
*   `SetID(id string)`: Inject unique ID.
*   `Render()`: Returns the `dom.Node` tree (Preferred).
*   `RenderHTML()`: Returns static HTML string (Legacy/Fallback).
*   `Children()`: Returns child components (Optional with BaseComponent).

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
2.  **Child Components**: **Recursive Lifecycle**. The library automatically renders and unmounts child components found in the `Node` tree.
3.  **State Updates**: **Component-Level Reactivity**. Calling `c.Update()` re-renders the component and replaces it in the DOM.
4.  **Builder API**: **Declarative vs Imperative**. We use a Builder pattern (`dom.Div`) to ensure type safety, correct HTML structure, and automated event wiring.
5.  **TinyGo Optimization**: **Slices vs Maps**. To minimize WASM binary size and improve performance, elements use `[]fmt.KeyValue` for attributes and `[]EventHandler` for events instead of Go maps. Linear scans are faster for typical attribute counts (< 10) and avoid the heavy map runtime support.
6.  **CSS Strategy**: **Global**. Standard CSS classes.
7.  **Routing**: **Single Page Root**. The "Index" is the root component.

## 7. API Design Philosophy (Q&A)

### Why `Get(id string)` and not `Get(c Component)`?
*   **Flexibility**: Often you need to manipulate a plain DOM element (like a container `<div>` or an input) that doesn't have a corresponding Go Component struct.
*   **Decoupling**: It allows the DOM to be treated as a collection of nodes, regardless of how they were created.

### Why `Render(parentID string, ...)`?
*   The target container (e.g., `<div id="app">`) usually exists in the static HTML or is part of a parent's template. It is rarely a Component itself.

### Decoupled Components
Components can define their own narrow interfaces to avoid depending on the full `tinywasm/dom.DOM` package.
*   *Example*: If a component only needs to update text, it can define `type TextUpdater interface { Get(id string) Element }` and accept that, making it easier to test and reuse.
