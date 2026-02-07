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

// Mount injects the component's HTML into the parent element and calls OnMount recursively.
func (d *domWasm) Mount(parentID string, component Component) error {
	parent, ok := d.Get(parentID)
	if !ok {
		return fmt.Errf("parent element not found: %s", parentID)
	}

	parent.SetHTML(component.RenderHTML())

	d.mountRecursive(component)
	return nil
}

func (d *domWasm) mountRecursive(c Component) {
	// Track the component being mounted so that event listeners registered
	// during OnMount are associated with this component ID for auto-cleanup.
	prevID := d.currentComponentID
	d.currentComponentID = c.ID()
	// Restore the previous ID after this component and its children are mounted
	defer func() { d.currentComponentID = prevID }()

	// Only call OnMount if component implements Mountable
	if mountable, ok := c.(Mountable); ok {
		mountable.OnMount()
	}

	// Recursively mount children
	for _, child := range c.Children() {
		if child != nil {
			d.mountRecursive(child)
		}
	}
}

// OnHashChange registers a listener for window.hashchange.
func (d *domWasm) OnHashChange(handler func(hash string)) {
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler(d.GetHash())
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "hashchange", fn)
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

// Unmount removes a component from the DOM and recursively cleans up children.
func (d *domWasm) Unmount(component Component) {
	d.unmountRecursive(component)

	// Remove the element from the DOM
	id := component.ID()
	el, ok := d.Get(id)
	if ok {
		el.Remove()
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

func (d *domWasm) unmountRecursive(c Component) {
	// Recursively cleanup children first
	for _, child := range c.Children() {
		if child != nil {
			d.unmountRecursive(child)
		}
	}

	// Call OnUnmount if component implements Mountable
	if mountable, ok := c.(Mountable); ok {
		mountable.OnUnmount()
	}

	d.cleanupListeners(c.ID())
}

// cleanupListeners releases all functions associated with the component ID.
func (d *domWasm) cleanupListeners(id string) {
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
			for i, ef := range d.eventFuncs {
				if ef.key == key {
					ef.fn.Release()
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
}
