# TinyDOM API Reference

## Global API

TinyDOM provides a global API for direct access to the DOM in WASM environments.

```go
// Get retrieves an element by its ID.
Get(id string) (Element, bool)

// Render injects a component into a parent element.
// If the component has an empty ID, it auto-generates a unique one (e.g., "tiny-1").
Render(parentID string, component Component) error

// Append injects a component AFTER the last child of the parent element.
Append(parentID string, component Component) error

// Hydrate attaches event listeners to existing HTML without re-rendering it.
// Use this for server-rendered components to enable interactivity without flicker.
Hydrate(parentID string, component Component) error

// Update re-renders a component in-place.
Update(component Component) error

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


## Identifiable Interface

Provides unique identification for components.

```go
type Identifiable interface {
	// GetID returns the unique identifier.
	GetID() string
	// SetID sets the unique identifier.
	SetID(id string)
}
```

type Component interface {
	Identifiable
	HTMLRenderer
	ChildProvider
}

// ViewRenderer provides a declarative way to render the component.
type ViewRenderer interface {
	Render() Node
}

// EventHandler represents a DOM event handler in the declarative builder.
type EventHandler struct {
	Name    string
	Handler func(Event)
}

// Node represents a DOM element in the declarative Builder API.
type Node struct {
	Tag      string
	Attrs    []fmt.KeyValue
	Events   []EventHandler
	Children []any
}

## Builder API

The Builder API provides a fluent, declarative way to create `Node` trees.

```go
// Add one or more children
Div().Add(
    Span().Text("Hello"),
    Button().Text("Click Me"),
)

// Auto-ID Injection
// When used in a ViewRenderer, the root node automatically gets the component's ID.
func (c *MyComp) Render() dom.Node {
    return dom.Div(). // ID(c.GetID()) is no longer required here
        Class("my-comp").
        Add(Span().Text("Content")).
        ToNode()
}
```

### Methods

- `ID(id string)`: Sets the element's ID.
- `Class(name string)`: Adds a CSS class.
- `Attr(key, val string)`: Sets a custom attribute.
- `Add(children ...any)`: Adds multiple children (Builders, Nodes, Components, or strings).
- `Text(text string)`: Adds a text node child.
- `OnClick(handler func(Event))`: Binds a click event.
- `ToNode()`: terminal operation that returns a `Node`.

> [!TIP]
> Use `dom.BaseComponent` to automatically implement the `Identifiable` interface in your structs.

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

**Key Change**: Components no longer receive a `DOM` instance as a parameter in `OnMount()`. Instead, they use the global `dom.Get()`, `dom.Render()`, etc.

### Backend-Only: CSS and JS Rendering

For SSR with styles and scripts, optionally implement these interfaces:

```go
//go:build !wasm

type CSSProvider interface {
	Component
	RenderCSS() string
}

type JSProvider interface {
	Component
	RenderJS() string
}
```

These methods are only called on the backend for server-side rendering.

## Access Control

The `AccessLevel` interface provides access control information for components.

```go
type AccessLevel interface {
	// AllowedRoles returns the list of allowed roles for a given action (e.g., 'r' for read).
	// Returning "*" grants access to everyone.
	AllowedRoles(action byte) []byte
}
```
