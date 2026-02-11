package dom

import (
	"github.com/tinywasm/fmt"
)

var (
	shared    = &tinyDOM{}
	instance  = newDom(shared)
	idCounter uint64
)

// tinyDOM contains shared functionality between backend and WASM implementations.
type tinyDOM struct {
	log func(v ...any)
}

// Get retrieves an element by its ID.
func Get(id string) (Reference, bool) {
	return instance.Get(id)
}

// generateID creates a unique ID for a component.
func generateID() string {
	idCounter++
	return fmt.Sprint(idCounter)
}

// Render injects a component into a parent element.
func Render(parentID string, component Component) error {
	if component.GetID() == "" {
		component.SetID(generateID())
	}
	return instance.Render(parentID, component)
}

// Append injects a component AFTER the last child of the parent element.
func Append(parentID string, component Component) error {
	if component.GetID() == "" {
		component.SetID(generateID())
	}
	return instance.Append(parentID, component)
}

// Hydrate attaches event listeners to existing HTML.
func Hydrate(parentID string, component Component) error {
	if component.GetID() == "" {
		// In hydration, we assume the ID matches what was rendered.
		// If not provided, we generate one, but this might not match server-rendered ID.
		// Ideally hydration requires consistent IDs.
		component.SetID(generateID())
	}
	return instance.Hydrate(parentID, component)
}

// Update re-renders a component.
func Update(component Component) error {
	return instance.Update(component)
}

// Unmount removes a component from the DOM.
func Unmount(component Component) {
	instance.Unmount(component)
}

// Log provides logging functionality.
func Log(v ...any) {
	instance.Log(v...)
}

// OnHashChange registers a hash change listener.
func OnHashChange(handler func(hash string)) {
	instance.OnHashChange(handler)
}

// GetHash gets the current hash.
func GetHash() string {
	return instance.GetHash()
}

// SetHash sets the current hash.
func SetHash(hash string) {
	instance.SetHash(hash)
}

// QueryAll query elements.
func QueryAll(selector string) []Reference {
	return instance.QueryAll(selector)
}

// SetLog sets the logging function.
func SetLog(log func(v ...any)) {
	shared.log = log
}

// injectComponentID sets the component ID on the root node if not already set.
func injectComponentID(n Node, id string) Node {
	for _, attr := range n.Attrs {
		if attr.Key == "id" {
			return n // Already has an ID, don't override
		}
	}
	n.Attrs = append([]fmt.KeyValue{{Key: "id", Value: id}}, n.Attrs...)
	return n
}

// Log provides logging functionality using the log function passed to New.
func (t *tinyDOM) Log(v ...any) {
	if t.log != nil {
		t.log(v...)
	}
}
