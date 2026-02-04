//go:build wasm

package dom

import (
	"syscall/js"

	"github.com/tinywasm/fmt"
)

// domWasm is the WASM implementation of the DOM interface.
type domWasm struct {
	*tinyDOM
	document     js.Value // Cached document object
	elementCache []struct {
		id  string
		val js.Value
	}
	eventFuncs []struct {
		key string
		fn  js.Func
	}
	componentListeners []struct {
		id   string
		keys []string
	}
	currentComponentID string // Tracks the component being mounted
}

// newDom returns a new instance of the domWasm.
func newDom(td *tinyDOM) DOM {
	return &domWasm{
		tinyDOM:  td,
		document: js.Global().Get("document"),
	}
}

// Get retrieves an element by ID from the cache or the DOM.
func (d *domWasm) Get(id string) (Element, bool) {
	// Linear search in cache
	for _, item := range d.elementCache {
		if item.id == id {
			return &elementWasm{
				val: item.val,
				dom: d,
				id:  id,
			}, true
		}
	}

	val := d.document.Call("getElementById", id)
	if val.IsNull() || val.IsUndefined() {
		d.Log("tinywasm/dom: element with id", id, "not found") // Optional logging
		return nil, false
	}

	// Append to cache
	d.elementCache = append(d.elementCache, struct {
		id  string
		val js.Value
	}{id, val})

	return &elementWasm{
		val: val,
		dom: d,
		id:  id,
	}, true
}

// Mount injects the component's HTML into the parent element and calls OnMount.
func (d *domWasm) Mount(parentID string, component Component) error {
	parent, ok := d.Get(parentID)
	if !ok {
		// Return a simple error instead of js.Error to avoid panics during formatting
		return fmt.Errf("parent element not found: %s", parentID)
	}

	d.currentComponentID = component.HandlerName()
	parent.SetHTML(component.RenderHTML())

	// Only call OnMount if component implements Mountable
	if mountable, ok := component.(Mountable); ok {
		mountable.OnMount()
	}

	d.currentComponentID = ""
	return nil
}

// OnHashChange registers a listener for window.hashchange.
func (d *domWasm) OnHashChange(handler func(hash string)) {
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler(d.GetHash())
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "hashchange", fn)
	// Track global event if we want cleanup, but here it's likely app-lifetime
}

// GetHash returns current window.location.hash.
func (d *domWasm) GetHash() string {
	return js.Global().Get("location").Get("hash").String()
}

// SetHash updates window.location.hash.
func (d *domWasm) SetHash(hash string) {
	js.Global().Get("location").Set("hash", hash)
}

// QueryAll performs a document.querySelectorAll and wraps the results.
func (d *domWasm) QueryAll(selector string) []Element {
	nodes := d.document.Call("querySelectorAll", selector)
	count := nodes.Length()
	elems := make([]Element, count)
	for i := 0; i < count; i++ {
		val := nodes.Index(i)
		id := val.Get("id").String()
		elems[i] = &elementWasm{
			val: val,
			dom: d,
			id:  id,
		}
	}
	return elems
}

// Unmount removes a component from the DOM and cleans up event listeners.
func (d *domWasm) Unmount(component Component) {
	// Only call OnUnmount if component implements Mountable
	if mountable, ok := component.(Mountable); ok {
		mountable.OnUnmount()
	}

	id := component.HandlerName()
	// Remove the element from the DOM
	el, ok := d.Get(id)
	if ok {
		el.Remove()
	}

	// Efficiently clean up listeners for this component
	// Find listeners for this component
	var listeners []string
	compIndex := -1
	for i, item := range d.componentListeners {
		if item.id == id {
			listeners = item.keys
			compIndex = i
			break
		}
	}

	if compIndex != -1 {
		// Release each function
		for _, key := range listeners {
			// Find and release the function in eventFuncs
			for i, ef := range d.eventFuncs {
				if ef.key == key {
					ef.fn.Release()
					// Remove from eventFuncs (swap and pop or just copy)
					// Since order doesn't matter much for internal storage, swap remove is faster
					lastIdx := len(d.eventFuncs) - 1
					d.eventFuncs[i] = d.eventFuncs[lastIdx]
					d.eventFuncs = d.eventFuncs[:lastIdx]
					break
				}
			}
		}

		// Remove from componentListeners
		lastIdx := len(d.componentListeners) - 1
		d.componentListeners[compIndex] = d.componentListeners[lastIdx]
		d.componentListeners = d.componentListeners[:lastIdx]
	}

	// Remove from cache
	for i, item := range d.elementCache {
		if item.id == id {
			lastIdx := len(d.elementCache) - 1
			d.elementCache[i] = d.elementCache[lastIdx]
			d.elementCache = d.elementCache[:lastIdx]
			break
		}
	}
}
