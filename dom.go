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

// generateID creates a unique ID for a component.
func generateID() string {
	idCounter++
	return fmt.Sprint(idCounter)
}

// Render injects a component into a parent element.
func Render(parentID string, component Component) error {
	return instance.Render(parentID, component)
}

// Append injects a component AFTER the last child of the parent element.
func Append(parentID string, component Component) error {
	return instance.Append(parentID, component)
}

// Update re-renders a component.
func Update(component Component) error {
	return instance.Update(component)
}

// Log provides logging functionality.
func Log(v ...any) {
	instance.Log(v...)
}

// Get retrieves an element by ID.
func Get(id string) (Reference, bool) {
	return instance.Get(id)
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

// SetLog sets the logging function.
func SetLog(log func(v ...any)) {
	shared.log = log
}

// injectComponentID sets the component ID on the root element if not already set.
func injectComponentID(el *Element, id string) {
	if el.id == "" {
		el.id = id
	}
}

// Log provides logging functionality using the log function passed to New.
func (t *tinyDOM) Log(v ...any) {
	if t.log != nil {
		t.log(v...)
	}
}
