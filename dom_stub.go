//go:build !wasm

package tinydom

// domStub is a no-op implementation of the DOM interface for non-WASM targets.
type domStub struct {
	log func(v ...any)
}

// newDom returns a new instance of the domStub.
func newDom(log func(v ...any)) DOM {
	return &domStub{log: log}
}

// Get returns a no-op element.
func (d *domStub) Get(id string) (Element, bool) {
	return &elementStub{}, true
}

// Mount does nothing on non-WASM targets.
func (d *domStub) Mount(parentID string, component Component) error {
	d.log("tinydom: Mount called on stub for parent", parentID)
	return nil
}

// Unmount does nothing on non-WASM targets.
func (d *domStub) Unmount(component Component) {
	d.log("tinydom: Unmount called on stub for", component.ID())
}
