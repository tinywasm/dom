# TinyDOM API Reference

## Global API

TinyDOM provides a global API for direct access to the DOM in WASM environments.

```go
// Get retrieves an element by its ID.
Get(id string) (Element, bool)

// Mount injects a component into a parent element.
Mount(parentID string, component Component) error

// Unmount removes a component from the DOM.
Unmount(component Component)

// Log provides logging functionality.
Log(v ...any)

// SetLog sets the logging function.
SetLog(log func(v ...any))
```

## Element Interface

The `Element` interface represents a DOM node with methods for content manipulation, styling, and event handling.

**📖 Full API Documentation**: See [`element.go`](../element.go) for complete interface definition with detailed examples.

### Key Features

All content methods (`SetText`, `SetHTML`, `AppendHTML`, `SetAttr`, `SetValue`) accept variadic arguments and support:
- **String concatenation** without spaces
- **Printf-style formatting** with `%` specifiers
- **Localized content** using `D.*` dictionary
- **Mixed types** (strings, numbers, etc.)

### Quick Examples

```go
// Simple concatenation
elem.SetText("Count: ", 42)              // -> "Count: 42"

// HTML with format strings
elem.SetHTML("<h1>%v</h1>", title)       // -> "<h1>My Title</h1>"

// Localized content
elem.SetText(D.Hello)                    // -> "Hello" (EN) or "Hola" (ES)

// Multiline HTML components
elem.SetHTML(`<div class='card'>
	<h2>%L</h2>
	<p>%v</p>
</div>`, D.Title, count)

// Attributes
elem.SetAttr("id", "item-", 42)          // -> id="item-42"
elem.SetAttr("href", "/page/", pageNum)  // -> href="/page/5"
```

For complete method signatures and more examples, see [`element.go`](../element.go).


## Event Interface

The `Event` interface wraps the native browser event to provide a safe, simplified API.

```go
type Event interface {
	// PreventDefault prevents the default action of the event.
	PreventDefault()

	// StopPropagation stops the event from bubbling up the DOM tree.
	StopPropagation()

	// TargetValue returns the value of the event's target element.
	// Useful for input, textarea, and select elements.
	TargetValue() string

	// TargetID returns the ID of the event's target element.
	TargetID() string
}
```


## Component Interface

The minimal interface that all components must implement for both SSR (backend) and WASM (frontend):

```go
type Component interface {
	// ID returns the unique identifier of the component's root element.
	ID() string

	// RenderHTML returns the full HTML string of the component.
	// The root element of this HTML MUST have the id returned by ID().
	RenderHTML() string
}
```

### WASM-Only: Mountable

For interactive components in the browser, implement the `Mountable` interface:

```go
//go:build wasm

type Mountable interface {
	Component

	// OnMount is called after the HTML has been injected into the DOM.
	// The component can now use the global API to bind events and interact with elements.
	OnMount()

	// OnUnmount is called before the component is removed from the DOM.
	OnUnmount()
}
```

**Key Change**: Components no longer receive a `DOM` instance as a parameter in `OnMount()`. Instead, they use the global `dom.Get()`, `dom.Mount()`, etc.

### Backend-Only: CSS and JS Rendering

For SSR with styles and scripts, optionally implement these interfaces:

```go
//go:build !wasm

type CSSRenderer interface {
	Component
	RenderCSS() string
}

type JSRenderer interface {
	Component
	RenderJS() string
}
```

These methods are only called on the backend for server-side rendering.
