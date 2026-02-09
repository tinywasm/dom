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

	// Lifecycle tracking (using slices to avoid map overhead)
	mountedComponents []struct {
		id   string
		comp Component
	}
	childrenMap []struct {
		parentID string
		childIDs []string
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
		// d.Log("tinywasm/dom: element with id", id, "not found")
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

// Render injects the component's content into the parent element.
func (d *domWasm) Render(parentID string, component Component) error {
	parent, ok := d.Get(parentID)
	if !ok {
		return fmt.Errf("parent element not found: %s", parentID)
	}

	// Generate ID if not set
	if component.GetID() == "" {
		component.SetID(generateID())
	}

	// Render to HTML and collect child components
	var children []Component
	var html string

	if vr, ok := component.(ViewRenderer); ok {
		html = d.renderToHTML(vr.Render(), &children)
	} else if el, ok := component.(*Builder); ok {
		html = d.renderToHTML(el.ToNode(), &children)
	} else {
		html = component.RenderHTML()
	}

	parent.SetHTML(html)

	// Update lifecycle maps
	d.trackComponent(component)
	d.trackChildren(component.GetID(), children)

	// Wire pending events
	d.wirePendingEvents()

	// Mount logic
	d.mountRecursive(component)
	for _, child := range children {
		d.mountRecursive(child)
	}

	return nil
}

// Update re-renders the component and replaces it in the DOM.
func (d *domWasm) Update(component Component) error {
	id := component.GetID()
	el, ok := d.Get(id)
	if !ok {
		return fmt.Errf("component element not found: %s", id)
	}

	// Clean up old children listeners/lifecycle
	d.cleanupChildren(id)

	// Clean up listeners for the component itself (will be re-bound if OnUpdate adds them, or if DSL adds them)
	// Note: DSL events are wired via wirePendingEvents.
	// Manual events in OnUpdate need currentComponentID.
	// Existing listeners on old element are gone from DOM, but need to be removed from Go map.
	d.cleanupListeners(id)

	var children []Component
	var html string

	if vr, ok := component.(ViewRenderer); ok {
		html = d.renderToHTML(vr.Render(), &children)
	} else if el, ok := component.(*Builder); ok {
		html = d.renderToHTML(el.ToNode(), &children)
	} else {
		html = component.RenderHTML()
	}

	// Replace the element in the DOM
	elWasm := el.(*elementWasm)
	elWasm.val.Set("outerHTML", html)

	// Clear element from cache as it was replaced
	d.removeFromElementCache(id)

	// Update lifecycle maps
	d.trackChildren(id, children)

	// Wire events (DSL)
	d.wirePendingEvents()

	// Set current component ID for OnUpdate and manual listeners
	prevID := d.currentComponentID
	d.currentComponentID = id
	defer func() { d.currentComponentID = prevID }()

	// Call OnUpdate hook if implemented
	if updatable, ok := component.(Updatable); ok {
		updatable.OnUpdate()
	}

	// Mount new children
	for _, child := range children {
		d.mountRecursive(child)
	}

	return nil
}

// Append injects the component's content after the last child of the parent element.
func (d *domWasm) Append(parentID string, component Component) error {
	parent, ok := d.Get(parentID)
	if !ok {
		return fmt.Errf("parent element not found: %s", parentID)
	}

	if component.GetID() == "" {
		component.SetID(generateID())
	}

	var children []Component
	var html string
	if vr, ok := component.(ViewRenderer); ok {
		html = d.renderToHTML(vr.Render(), &children)
	} else if el, ok := component.(*Builder); ok {
		html = d.renderToHTML(el.ToNode(), &children)
	} else {
		html = component.RenderHTML()
	}

	parent.AppendHTML(html)

	d.trackComponent(component)
	d.trackChildren(component.GetID(), children)
	d.wirePendingEvents()

	d.mountRecursive(component)
	for _, child := range children {
		d.mountRecursive(child)
	}
	return nil
}

// Hydrate attaches event listeners to existing HTML.
func (d *domWasm) Hydrate(parentID string, component Component) error {
	_, ok := d.Get(parentID)
	if !ok {
		return fmt.Errf("parent element not found: %s", parentID)
	}
	d.trackComponent(component)
	d.mountRecursive(component)
	return nil
}

// Unmount removes a component from the DOM and recursively cleans up children.
func (d *domWasm) Unmount(component Component) {
	d.unmountRecursive(component)

	// Remove the element from the DOM
	id := component.GetID()
	el, ok := d.Get(id)
	if ok {
		el.Remove()
	}

	d.removeFromElementCache(id)
	d.untrackComponent(id)
}

func (d *domWasm) renderToHTML(n Node, comps *[]Component) string {
	// If the node has events but no ID, generate one
	id := ""
	for _, attr := range n.Attrs {
		if attr.Key == "id" {
			id = attr.Value
			break
		}
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
			s += d.renderToHTML(v, comps)
		case string:
			s += v
		case Component:
			*comps = append(*comps, v)
			// Ensure ID
			if v.GetID() == "" {
				v.SetID(generateID())
			}

			if vr, ok := v.(ViewRenderer); ok {
				s += d.renderToHTML(vr.Render(), comps)
			} else if el, ok := v.(*Builder); ok {
				s += d.renderToHTML(el.ToNode(), comps)
			} else {
				s += v.RenderHTML()
			}
		default:
			s += fmt.Sprint(v)
		}
	}
	s += "</" + n.Tag + ">"
	return s
}

func (d *domWasm) mountRecursive(c Component) {
	prevID := d.currentComponentID
	d.currentComponentID = c.GetID()
	defer func() { d.currentComponentID = prevID }()

	if mountable, ok := c.(Mountable); ok {
		mountable.OnMount()
	}

	// Mount children (for HTMLRenderer primarily)
	for _, child := range c.Children() {
		if child != nil {
			d.mountRecursive(child)
		}
	}

	// Children tracked in maps are mounted in Render/Update loop
}

func (d *domWasm) unmountRecursive(c Component) {
	// Cleanup children first
	// 1. From Children() interface
	for _, child := range c.Children() {
		if child != nil {
			d.unmountRecursive(child)
		}
	}

	// 2. From tracked children map
	id := c.GetID()
	var childIDs []string
	for i, item := range d.childrenMap {
		if item.parentID == id {
			childIDs = item.childIDs
			// Remove from map
			lastIdx := len(d.childrenMap) - 1
			d.childrenMap[i] = d.childrenMap[lastIdx]
			d.childrenMap = d.childrenMap[:lastIdx]
			break
		}
	}

	for _, childID := range childIDs {
		// Find component instance
		var childComp Component
		for i, item := range d.mountedComponents {
			if item.id == childID {
				childComp = item.comp
				// Remove from mounted
				lastIdx := len(d.mountedComponents) - 1
				d.mountedComponents[i] = d.mountedComponents[lastIdx]
				d.mountedComponents = d.mountedComponents[:lastIdx]
				break
			}
		}
		if childComp != nil {
			d.unmountRecursive(childComp)
		} else {
			// If instance not found, just cleanup listeners
			d.cleanupListeners(childID)
		}
	}

	if unmountable, ok := c.(Unmountable); ok {
		unmountable.OnUnmount()
	}

	d.cleanupListeners(c.GetID())
}

func (d *domWasm) cleanupChildren(parentID string) {
	// Similar to unmountRecursive but only for tracked children
	var childIDs []string
	for i, item := range d.childrenMap {
		if item.parentID == parentID {
			childIDs = item.childIDs
			lastIdx := len(d.childrenMap) - 1
			d.childrenMap[i] = d.childrenMap[lastIdx]
			d.childrenMap = d.childrenMap[:lastIdx]
			break
		}
	}

	for _, childID := range childIDs {
		var childComp Component
		for i, item := range d.mountedComponents {
			if item.id == childID {
				childComp = item.comp
				lastIdx := len(d.mountedComponents) - 1
				d.mountedComponents[i] = d.mountedComponents[lastIdx]
				d.mountedComponents = d.mountedComponents[:lastIdx]
				break
			}
		}
		if childComp != nil {
			d.unmountRecursive(childComp)
		} else {
			d.cleanupListeners(childID)
		}
	}
}

// Helpers for state management

func (d *domWasm) trackComponent(c Component) {
	id := c.GetID()
	for _, item := range d.mountedComponents {
		if item.id == id {
			return // Already tracked
		}
	}
	d.mountedComponents = append(d.mountedComponents, struct{id string; comp Component}{id, c})
}

func (d *domWasm) untrackComponent(id string) {
	for i, item := range d.mountedComponents {
		if item.id == id {
			lastIdx := len(d.mountedComponents) - 1
			d.mountedComponents[i] = d.mountedComponents[lastIdx]
			d.mountedComponents = d.mountedComponents[:lastIdx]
			break
		}
	}
}

func (d *domWasm) trackChildren(parentID string, children []Component) {
	childIDs := make([]string, len(children))
	for i, c := range children {
		childIDs[i] = c.GetID()
		d.trackComponent(c)
	}

	// Check if entry exists
	found := false
	for i, item := range d.childrenMap {
		if item.parentID == parentID {
			d.childrenMap[i].childIDs = childIDs
			found = true
			break
		}
	}
	if !found {
		d.childrenMap = append(d.childrenMap, struct{parentID string; childIDs []string}{parentID, childIDs})
	}
}

func (d *domWasm) removeFromElementCache(id string) {
	for i, item := range d.elementCache {
		if item.id == id {
			lastIdx := len(d.elementCache) - 1
			d.elementCache[i] = d.elementCache[lastIdx]
			d.elementCache = d.elementCache[:lastIdx]
			break
		}
	}
}

func (d *domWasm) wirePendingEvents() {
	for _, pe := range d.pendingEvents {
		if el, ok := d.Get(pe.id); ok {
			el.On(pe.name, pe.handler)
		}
	}
	d.pendingEvents = nil
}

func (d *domWasm) cleanupListeners(id string) {
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
		lastIdx := len(d.componentListeners) - 1
		d.componentListeners[compIndex] = d.componentListeners[lastIdx]
		d.componentListeners = d.componentListeners[:lastIdx]
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
