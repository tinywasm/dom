# TinyDOM API Reference

## Core Interface

The `DOM` interface is the main entry point for interacting with the browser. It is designed to be injected into your components.

```go
package tinydom

type DOM interface {
	// Get retrieves an element by its ID.
	// It uses an internal cache to avoid repeated DOM lookups.
	Get(id string) (Element, bool)

	// Mount injects a component into a parent element.
	// 1. It calls component.RenderHTML()
	// 2. It sets the InnerHTML of the parent element (found by parentID)
	// 3. It calls component.OnMount() to bind events
	Mount(parentID string, component Component) error

	// Unmount removes a component from the DOM (by clearing the parent's HTML or removing the node)
	// and cleans up any event listeners registered via the Element interface.
	Unmount(component Component)
}
```

## Element Interface

The `Element` interface represents a DOM node. It provides methods for direct manipulation and event binding.

```go
type Element interface {
	// --- Content ---

	// SetText sets the text content of the element.
	SetText(text string)

	// SetHTML sets the inner HTML of the element.
	SetHTML(html string)

	// AppendHTML adds HTML to the end of the element's content.
	// Useful for adding items to a list without re-rendering the whole list.
	AppendHTML(html string)

	// Remove removes the element from the DOM.
	Remove()

	// --- Attributes & Classes ---

	// AddClass adds a CSS class to the element.
	AddClass(class string)

	// RemoveClass removes a CSS class from the element.
	RemoveClass(class string)

	// ToggleClass toggles a CSS class.
	ToggleClass(class string)

	// SetAttr sets an attribute value.
	SetAttr(key, value string)

	// GetAttr retrieves an attribute value.
	GetAttr(key string) string

	// RemoveAttr removes an attribute.
	RemoveAttr(key string)

	// --- Forms ---

	// Value returns the current value of an input/textarea/select.
	Value() string

	// SetValue sets the value of an input/textarea/select.
	SetValue(value string)

	// --- Events ---

	// Click registers a click event handler.
	// The handler is automatically tracked and removed when the component is unmounted.
	Click(handler func(event Event))

	// On registers a generic event handler (e.g., "change", "input", "keydown").
	On(eventType string, handler func(event Event))
    
    // Focus sets focus to the element.
    Focus()
}
```

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
	// The DOM instance is passed so the component can bind events and interact with elements.
	OnMount(dom DOM)

	// OnUnmount is called before the component is removed from the DOM.
	OnUnmount()
}
```

**Key Change**: Components now receive the `DOM` instance as a parameter in `OnMount()` instead of storing it as a field.

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
