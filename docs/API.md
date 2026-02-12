# TinyDOM API Reference

## Global API

TinyDOM provides a global API for direct access to the DOM in WASM environments.

```go
// Render injects a component into a parent element.
Render(parentID string, component Component) error

// Append injects a component AFTER the last child of the parent element.
Append(parentID string, component Component) error

// Update re-renders a component in-place.
Update(component Component) error

// Routing (hash-based)
OnHashChange(handler func(hash string))
GetHash() string
SetHash(hash string)
```

## Reference Interface

The `Reference` interface represents a live DOM node in the browser. It provides methods for reading state and basic interaction.

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


// Component is the unified interface for all components.
type Component interface {
	GetID() string
	SetID(id string)
	RenderHTML() string
	Children() []Component
}

// ViewRenderer provides a declarative way to render the component.
type ViewRenderer interface {
	Render() *Element
}

// EventHandler represents a DOM event handler in the declarative builder.
type EventHandler struct {
	Name    string
	Handler func(Event)
}

// Element represents a DOM element in the declarative Builder API.
// It is the unified type for building and rendering.

## Element API (JSX-like Builder)

The `Element` API provides a declarative way to create element trees with concise nesting and typed form elements.

### Basic Containers
All container factories accept `(children ...any)`. Children can be `*Element`, `Component`, `string`, or `any` (which uses `fmt.Sprint`).

```go
Div(
    H1("Title"),
    P("This is a paragraph with ", Strong("bold text"), "."),
    Ul(
        Li("Item 1"),
        Li("Item 2"),
    ),
)
```

### Strongly Typed Form Elements
Concrete types like `*InputEl`, `*FormEl`, `*SelectEl`, and `*TextareaEl` provide semantic methods that preserve the typed chain.

```go
Form(
    Text("username", "Enter username").Required(),
    Email("email").Placeholder("Your email").Class("field"),
    Password("pwd"),
    Select("role",
        Option("admin", "Administrator"),
        SelectedOption("user", "User"),
    ),
    Textarea("bio").Rows(5),
    Button("Submit").Attr("type", "submit"),
).Action("/api/login").Method("POST")
```

### Void Elements
Void elements render correctly without closing tags (e.g., `<br>` instead of `<br></br>`).
- `Br()`, `Hr()`
- `Img(src, alt)`
- All `Input` types (renders as `<input type='...'>`)

### Common Chaining Methods
Available on all elements (shadowed on typed elements to preserve the chain):
- `ID(id string)`: Sets the element's ID.
- `Class(names ...string)`: Adds one or more CSS classes.
- `Attr(key, val string)`: Sets a custom attribute.
- `On(eventType string, handler func(Event))`: Binds an event handler.
- `Add(children ...any)`: Adds children dynamically.

> [!TIP]
> Embed `*dom.Element` in your structs to automatically implement the `Component` interface and gain access to lifecycle methods.

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
