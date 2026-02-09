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
	pendingEvents      []struct {
		id      string
		name    string
		handler func(Event)
	}
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

	var val js.Value
	switch id {
	case "body":
		val = d.document.Get("body")
	case "head":
		val = d.document.Get("head")
	default:
		val = d.document.Call("getElementById", id)
	}

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

// Render injects the component's content into the parent element and calls OnMount recursively.
func (d *domWasm) Render(parentID string, component Component) error {
	parent, ok := d.Get(parentID)
	if !ok {
		return fmt.Errf("parent element not found: %s", parentID)
	}

	html := ""
	if vr, ok := component.(ViewRenderer); ok {
		html = d.renderToHTML(vr.Render())
	} else {
		html = component.RenderHTML()
	}

	parent.SetHTML(html)

	// Wire events
	for _, pe := range d.pendingEvents {
		if el, ok := d.Get(pe.id); ok {
			el.On(pe.name, pe.handler)
		}
	}
	d.pendingEvents = nil

	d.mountRecursive(component)
	return nil
}

// Append injects the component's content after the last child of the parent element.
func (d *domWasm) Append(parentID string, component Component) error {
	parent, ok := d.Get(parentID)
	if !ok {
		return fmt.Errf("parent element not found: %s", parentID)
	}

	html := ""
	if vr, ok := component.(ViewRenderer); ok {
		html = d.renderToHTML(vr.Render())
	} else {
		html = component.RenderHTML()
	}

	parent.AppendHTML(html)

	// Wire events
	for _, pe := range d.pendingEvents {
		if el, ok := d.Get(pe.id); ok {
			el.On(pe.name, pe.handler)
		}
	}
	d.pendingEvents = nil

	d.mountRecursive(component)
	return nil
}

func (d *domWasm) renderToHTML(n Node) string {
	// If the node has events but no ID, generate one
	id := ""
	for i, attr := range n.Attrs {
		if attr.Key == "id" {
			id = attr.Value
			break
		}
		_ = i
	}

	if len(n.Events) > 0 && id == "" {
		id = "auto-" + generateID()
		n.Attrs = append(n.Attrs, fmt.KeyValue{Key: "id", Value: id})
	}

	for _, ev := range n.Events {
		d.pendingEvents = append(d.pendingEvents, struct {
			id      string
			name    string
			handler func(Event)
		}{id, ev.Name, ev.Handler})
	}

	s := "<" + n.Tag
	for _, attr := range n.Attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
	s += ">"
	for _, child := range n.Children {
		switch v := child.(type) {
		case Node:
			s += d.renderToHTML(v)
		case string:
			s += v
		case Component:
			// For components, we just render their placeholder
			// the recursive mount will handle their own Render/OnMount
			s += v.RenderHTML()
		}
	}
	s += "</" + n.Tag + ">"
	return s
}

// Hydrate attaches event listeners to existing HTML.
func (d *domWasm) Hydrate(parentID string, component Component) error {
	_, ok := d.Get(parentID)
	if !ok {
		return fmt.Errf("parent element not found: %s", parentID)
	}

	// We don't call parent.SetHTML(component.RenderHTML())
	// We just activate the lifecycle
	d.mountRecursive(component)
	return nil
}

// Update re-renders the component and replaces it in the DOM.
func (d *domWasm) Update(component Component) error {
	id := component.ID()
	el, ok := d.Get(id)
	if !ok {
		return fmt.Errf("component element not found: %s", id)
	}

	html := ""
	if vr, ok := component.(ViewRenderer); ok {
		html = d.renderToHTML(vr.Render())
	} else {
		html = component.RenderHTML()
	}

	// Replace the element in the DOM
	elWasm := el.(*elementWasm)
	elWasm.val.Set("outerHTML", html)

	// Since outerHTML replaced the element, we need to clear it from cache
	// so that the next Get(id) retrieves the new one.
	for i, item := range d.elementCache {
		if item.id == id {
			lastIdx := len(d.elementCache) - 1
			d.elementCache[i] = d.elementCache[lastIdx]
			d.elementCache = d.elementCache[:lastIdx]
			break
		}
	}

	// Wire events
	for _, pe := range d.pendingEvents {
		if el, ok := d.Get(pe.id); ok {
			el.On(pe.name, pe.handler)
		}
	}
	d.pendingEvents = nil

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
