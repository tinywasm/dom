//go:build wasm

package tinydom

import (
	"syscall/js"
)

// domWasm is the WASM implementation of the DOM interface.
type domWasm struct {
	log                func(v ...any)
	elementCache       map[string]js.Value
	eventFuncs         map[string]js.Func
	componentListeners map[string][]string // Maps component ID to a list of its event keys
	currentComponentID string            // Tracks the component being mounted
}

// newDom returns a new instance of the domWasm.
func newDom(log func(v ...any)) DOM {
	return &domWasm{
		log:                log,
		elementCache:       make(map[string]js.Value),
		eventFuncs:         make(map[string]js.Func),
		componentListeners: make(map[string][]string),
	}
}

// Get retrieves an element by ID from the cache or the DOM.
func (d *domWasm) Get(id string) (Element, bool) {
	if val, ok := d.elementCache[id]; ok {
		return &elementWasm{
			Value: val,
			dom:   d,
			id:    id,
		}, true
	}

	doc := js.Global().Get("document")
	val := doc.Call("getElementById", id)
	if val.IsNull() || val.IsUndefined() {
		d.log("tinydom: element with id", id, "not found")
		return nil, false
	}

	d.elementCache[id] = val
	return &elementWasm{
		Value: val,
		dom:   d,
		id:    id,
	}, true
}

// Mount injects the component's HTML into the parent element and calls OnMount.
func (d *domWasm) Mount(parentID string, component Component) error {
	parent, ok := d.Get(parentID)
	if !ok {
		d.log("tinydom: parent element with id", parentID, "not found for mounting")
		return &js.Error{Value: js.ValueOf("parent element not found")}
	}
	parent.SetHTML(component.RenderHTML())

	// Save the previous component ID and restore it after this mount is complete.
	// This correctly handles nested component mounting.
	previousComponentID := d.currentComponentID
	d.currentComponentID = component.ID()
	defer func() {
		d.currentComponentID = previousComponentID
	}()

	component.OnMount()

	return nil
}

// Unmount removes an element and its associated event listeners.
func (d *domWasm) Unmount(component Component) {
	component.OnUnmount()

	id := component.ID()
	// Remove the element from the DOM
	el, ok := d.Get(id)
	if ok {
		el.Remove()
	}

	// Efficiently clean up listeners for this component
	if listenerKeys, ok := d.componentListeners[id]; ok {
		for _, key := range listenerKeys {
			if fn, ok := d.eventFuncs[key]; ok {
				fn.Release()
				delete(d.eventFuncs, key)
			}
		}
		delete(d.componentListeners, id)
	}

	// Remove from cache
	delete(d.elementCache, id)
}
