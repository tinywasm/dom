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
func (d *domBackend) Get(id string) (Element, bool) {
	return nil, false
}

// Mount is not implemented for backend.
func (d *domBackend) Mount(parentID string, component Component) error {
	return fmt.Err("Mount is not implemented for backend")
}

// Unmount is not implemented for backend.
func (d *domBackend) Unmount(component Component) {
}

func (d *domBackend) OnHashChange(handler func(hash string)) {}
func (d *domBackend) GetHash() string                        { return "" }
func (d *domBackend) SetHash(hash string)                    {}
func (d *domBackend) QueryAll(selector string) []Element     { return nil }
