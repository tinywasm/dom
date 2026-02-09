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
func Get(id string) (Element, bool) {
	return instance.Get(id)
}

// generateID creates a unique ID for a component.
func generateID() string {
	idCounter++
	return fmt.Sprint(idCounter)
}

// Render injects a component into a parent element.
func Render(parentID string, component Component) error {
	if component.ID() == "" {
		component.SetID(generateID())
	}
	return instance.Render(parentID, component)
}

// Append injects a component AFTER the last child of the parent element.
func Append(parentID string, component Component) error {
	if component.ID() == "" {
		component.SetID(generateID())
	}
	return instance.Append(parentID, component)
}

// Mount is an alias for Render for backward compatibility.
// Deprecated: use Render instead.
func Mount(parentID string, component Component) error {
	return Render(parentID, component)
}

// Hydrate attaches event listeners to existing HTML.
func Hydrate(parentID string, component Component) error {
	if component.ID() == "" {
		// In hydration, we expect the ID to be there, but if not, we must set it
		// to match what the server theoretically rendered (though ideally the component
		// state should already have the ID).
		// For now, let's allow it to generate if empty, but usually it should be set.
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
func QueryAll(selector string) []Element {
	return instance.QueryAll(selector)
}

// SetLog sets the logging function.
func SetLog(log func(v ...any)) {
	shared.log = log
}

// Log provides logging functionality using the log function passed to New.
func (t *tinyDOM) Log(v ...any) {
	if t.log != nil {
		t.log(v...)
	}
}
