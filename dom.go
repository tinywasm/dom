package tinydom

// DOM is the main entry point for interacting with the browser.
// It is designed to be injected into your components.
type DOM interface {
	// Get retrieves an element by its ID.
	// It uses an internal cache to avoid repeated DOM lookups.
	// Returns the element and a boolean indicating if it was found.
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

// Component is the contract for your UI parts.
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

	// OnUnmount is called before the component is removed from the DOM.
	// Use this to perform cleanup, such as clearing timers or releasing resources.
	OnUnmount()
}
