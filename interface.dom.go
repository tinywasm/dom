package dom

import "github.com/tinywasm/fmt"

// DOM is the main entry point for interacting with the browser.
// It is designed to be injected into your components.
type DOM interface {
	// Get retrieves an element by its ID.
	// It uses an internal cache to avoid repeated DOM lookups.
	// Returns the element and a boolean indicating if it was found.
	Get(id string) (Element, bool)

	// Render injects a component into a parent element.
	// 1. It calls component.RenderHTML() (or component.Render() if available)
	// 2. It sets the content of the parent element (found by parentID)
	// 3. It calls component.OnMount() to bind events
	Render(parentID string, component Component) error

	// Append injects a component AFTER the last child of the parent element.
	// Useful for dynamic lists.
	Append(parentID string, component Component) error

	// Hydrate attaches event listeners to existing HTML without re-rendering it.
	Hydrate(parentID string, component Component) error

	// OnHashChange registers a listener for URL hash changes.
	OnHashChange(handler func(hash string))

	// GetHash returns the current URL hash (e.g., "#help").
	GetHash() string

	// SetHash updates the URL hash.
	SetHash(hash string)

	// QueryAll finds all elements matching a CSS selector.
	QueryAll(selector string) []Element

	// Unmount removes a component from the DOM (by clearing the parent's HTML or removing the node)
	// and cleans up any event listeners registered via the Element interface.
	Unmount(component Component)

	// Update re-renders the component in its current position in the DOM.
	Update(component Component) error

	// Log provides logging functionality using the log function passed to New.
	Log(v ...any)
}

// HTMLRenderer renders the component's HTML structure
type HTMLRenderer interface {
	RenderHTML() string
}

// ChildProvider returns the child components of a component.
type ChildProvider interface {
	// Children returns the child components.
	Children() []Component
}

// Identifiable provides a unique identifier for a component.
type Identifiable interface {
	ID() string
	SetID(id string)
}

// ViewRenderer returns a Node tree for declarative UI.
type ViewRenderer interface {
	Render() Node
}

// Component is the minimal interface for components.
// All components must implement this for both SSR (backend) and WASM (frontend).
type Component interface {
	Identifiable
	HTMLRenderer
	ChildProvider
}

// EventHandler represents a DOM event handler in the declarative builder.
type EventHandler struct {
	Name    string
	Handler func(Event)
}

// Node represents a DOM node in the declarative builder.
type Node struct {
	Tag      string
	Attrs    []fmt.KeyValue
	Events   []EventHandler
	Children []any // Can be Node, string, or Component
}
