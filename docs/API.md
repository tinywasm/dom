# TinyDOM API Reference

## Core Interface

The `DOM` interface is the main entry point for interacting with the browser. It is designed to be injected into your components.

```go
package tinydom

type DOM interface {
	// Get retrieves an element by its ID.
	// It uses an internal cache to avoid repeated DOM lookups.
	Get(id string) Element

	// Mount injects a component into a parent element.
	// 1. It calls component.RenderHTML()
	// 2. It sets the InnerHTML of the parent element (found by parentID)
	// 3. It calls component.OnMount() to bind events
	Mount(parentID string, component Component) error

	// Unmount removes a component from the DOM (by clearing the parent's HTML or removing the node)
	// and cleans up any event listeners registered via the Element interface.
	Unmount(id string)
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
	Click(handler func())

	// On registers a generic event handler (e.g., "change", "input", "keydown").
	On(eventType string, handler func())
    
    // Focus sets focus to the element.
    Focus()
}
```

## Component Interface

Any struct can be a component if it implements this interface.

```go
type Component interface {
	// ID returns the unique identifier of the component's root element.
	ID() string

	// RenderHTML returns the full HTML string of the component.
	// The root element of this HTML MUST have the id returned by ID().
	RenderHTML() string

	// OnMount is called after the HTML has been injected into the DOM.
	// Use this to:
	// 1. Get references to elements via dom.Get()
	// 2. Bind event listeners
	// 3. Initialize child components
	OnMount()
}
```
