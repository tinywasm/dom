package dom

var (
	shared   = &tinyDOM{}
	instance = newDom(shared)
)

// tinyDOM contains shared functionality between backend and WASM implementations.
type tinyDOM struct {
	log func(v ...any)
}

// Get retrieves an element by its ID.
func Get(id string) (Element, bool) {
	return instance.Get(id)
}

// Mount injects a component into a parent element.
func Mount(parentID string, component Component) error {
	return instance.Mount(parentID, component)
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
