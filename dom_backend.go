//go:build !wasm

package dom

import "github.com/tinywasm/fmt"

// domBackend is a stub implementation for non-WASM environments (e.g., SSR).
type domBackend struct {
	*tinyDOM
}

// newDom returns a new instance of the domBackend.
func newDom(td *tinyDOM) DOM {
	return &domBackend{
		tinyDOM: td,
	}
}

// Get is not implemented for backend.
func (d *domBackend) Get(id string) (Reference, bool) {
	return nil, false
}

// Render is not implemented for backend.
func (d *domBackend) Render(parentID string, component Component) error {
	return fmt.Errf("Render to parent is not supported on backend. Use RenderHTML() directly on component.")
}

// Append is not implemented for backend.
func (d *domBackend) Append(parentID string, component Component) error {
	return fmt.Errf("Append not supported in backend/stub")
}

// Hydrate is not implemented for backend.
func (d *domBackend) Hydrate(parentID string, component Component) error {
	return fmt.Errf("Hydrate is not implemented for backend")
}

// Update is not implemented for backend.
func (d *domBackend) Update(component Component) error {
	return fmt.Errf("Update is not implemented for backend")
}

// Unmount is not implemented for backend.
func (d *domBackend) Unmount(component Component) {
}

func (d *domBackend) OnHashChange(handler func(hash string)) {}
func (d *domBackend) GetHash() string                        { return "" }
func (d *domBackend) SetHash(hash string)                    {}
func (d *domBackend) QueryAll(selector string) []Reference   { return nil }
