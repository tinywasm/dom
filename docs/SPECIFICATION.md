# TinyDOM Specification

## 1. Philosophy & Goals

*   **Minimalist**: A thin wrapper over the browser DOM, optimized for TinyGo/WASM.
*   **No `syscall/js` exposure**: Components interact *only* with the TinyDOM Go interface.
*   **Zero StdLib**: Uses `tinystring` for string manipulation to keep binary size small.
*   **Direct & Cached**: Instead of a heavy VDOM tree diffing algorithm, it uses an **ID-based caching mechanism**. It maps Go IDs to browser DOM nodes to minimize JS calls.
*   **Standard HTML/CSS**: Components define their structure and style via standard string returns (`RenderHTML`, `RenderCSS`).
*   **Dependency Injection**: The `DOM` is injected into components, not imported globally.
*   **Auto ID Management**: The library can automatically assign unique IDs to components during `Mount`.
*   **Direct State Updates**: State changes update the DOM directly (e.g., `SetText`) to preserve focus and performance, rather than re-rendering the entire component.

## 2. Architecture

### The "Virtual" DOM (Cache Layer)
The "Virtual DOM" in this context is a lightweight state manager that:
1.  Maintains a map of `ID -> JS Reference`.
2.  Tracks active event listeners for cleanup.
3.  Handles mounting/unmounting of HTML strings into the actual DOM.

### Child Component Strategy (Manual Cascade)
To keep the framework minimal, parent components are responsible for:
1.  Concatenating child HTML strings in their own `RenderHTML`.
2.  Calling child `OnMount` methods within their own `OnMount`.
This avoids complex recursive mounting logic in the core library.

### Component Contract
A component is a Go struct that:
1.  Implements the `Identifiable` interface (`ID()` and `SetID()`).
2.  Implements methods to return its HTML/CSS.
3.  Manages its own events via the global API.

## 3. API Overview

> **Note:** This section provides a high-level overview. For the exact interface definitions and method signatures, please refer to the **[API Reference](API.md)**.

The architecture relies on three core interfaces:

### DOM Interface
The entry point injected into components. It handles:
*   **Caching**: `Get(id)` returns a cached `Element`.
*   **Lifecycle**: `Mount` injects HTML and binds events; `Unmount` cleans them up.

### Element Interface
Represents a DOM node. It provides methods for:
*   **Content**: `SetText`, `SetHTML`, `AppendHTML`.
*   **Attributes**: `AddClass`, `SetAttr`, `Value`.
*   **Events**: `Click`, `On` (with automatic cleanup).
*   **Manipulation**: `Remove` (for dynamic lists).

### Component Interface
The contract for UI parts:
*   `ID()`: Unique identifier.
*   `SetID(id string)`: Inject unique ID.
*   `RenderHTML()`: Returns the static HTML string.
*   `OnMount()`: Binds logic after the HTML is in the DOM.


## 4. Usage Example

> **See [Creating Components](COMPONENTS.md)** for detailed code examples, including basic counters and nested component structures.

## 5. Build Tags Strategy

*   **`dom_wasm.go`**: Implementation using `syscall/js` (hidden).
*   **`dom_stub.go`**: (`!wasm`) No-op implementation for server-side compilation safety.
*   **`css_loader.go`**: (`!wasm`) Logic to extract `RenderCSS()` from components and bundle/serve it.

## 6. Key Design Decisions

1.  **ID Management**: **Automatic**. `dom.Mount` handles unique ID generation for you.
2.  **Child Components**: **Manual Cascade**. Parent components explicitly include child HTML.
3.  **State Updates**: **Direct Manipulation**. Components update specific DOM elements directly.
4.  **Dynamic Lists**: **Node Manipulation**. To efficiently handle lists (add/remove items) without re-rendering the entire parent list, the API provides methods to append HTML and remove specific nodes.
    *   *Why?* `SetHTML` replaces all content. To remove just one item from a list of 100, we need `Element.Remove()` rather than re-generating the HTML for the remaining 99 items.
5.  **CSS Strategy**: **Global**. Standard CSS classes.
6.  **Routing**: **Single Page Root**. The "Index" is the root component. Page changes are handled by unmounting the current view component and mounting the new one into a main container.

## 7. API Design Philosophy (Q&A)

### Why `Get(id string)` and not `Get(c Component)`?
*   **Flexibility**: Often you need to manipulate a plain DOM element (like a container `<div>` or an input) that doesn't have a corresponding Go Component struct.
*   **Decoupling**: It allows the DOM to be treated as a collection of nodes, regardless of how they were created.

### Why `Mount(parentID string, ...)`?
*   The target container (e.g., `<div id="app">`) usually exists in the static HTML or is part of a parent's template. It is rarely a Component itself.

### Decoupled Components
Components can define their own narrow interfaces to avoid depending on the full `tinywasm/dom.DOM` package.
*   *Example*: If a component only needs to update text, it can define `type TextUpdater interface { Get(id string) Element }` and accept that, making it easier to test and reuse.
