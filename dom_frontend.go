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
		val js.Value // The element where listener is attached
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
func (d *domWasm) Get(id string) (Reference, bool) {
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

// getElement resolves a parentID to a js.Value, handling special cases like "body" and "head".
func (d *domWasm) getElement(id string) js.Value {
	switch id {
	case "body":
		return d.document.Get("body")
	case "head":
		return d.document.Get("head")
	default:
		return d.document.Call("getElementById", id)
	}
}

// Render injects the component's content into the parent element.
func (d *domWasm) Render(parentID string, component Component) error {
	if d.document.IsNull() || d.document.IsUndefined() {
		return fmt.Errf("document not found")
	}
	// Generate ID if not set
	if component.GetID() == "" {
		component.SetID(generateID())
	}

	// Render to HTML and collect child components
	var children []Component
	var html string

	if vr, ok := component.(ViewRenderer); ok {
		root := vr.Render()
		injectComponentID(root, component.GetID())
		html = d.renderToHTML(root, &children)
	} else if en, ok := component.(elementNode); ok {
		html = d.renderToHTML(en.AsElement(), &children)
	} else if el, ok := component.(*Element); ok {
		html = d.renderToHTML(el, &children)
	} else {
		html = component.RenderHTML()
	}

	parent := d.getElement(parentID)
	if parent.IsNull() || parent.IsUndefined() {
		return fmt.Errf("parent element not found: %s", parentID)
	}

	// Clean up any existing components in this parent before wiping content
	d.cleanupChildren(parentID)

	parent.Set("innerHTML", html)

	// Update lifecycle maps
	d.trackComponent(component)
	d.trackChildren(component.GetID(), children)

	// Set current component ID for event wiring
	prevID := d.currentComponentID
	d.currentComponentID = component.GetID()
	d.wirePendingEvents()
	d.currentComponentID = prevID

	// Mount logic
	d.mountRecursive(component)
	for _, child := range children {
		d.mountRecursive(child)
	}

	return nil
}

// Update re-renders the component and replaces it in the DOM.
func (d *domWasm) Update(component Component) error {
	if d.document.IsNull() || d.document.IsUndefined() {
		return fmt.Errf("document not found")
	}
	id := component.GetID()

	// Resolve the full outer component from tracked references.
	// This fixes Go embedding: Element.Update() passes *Element,
	// but mountedComponents stores the original *Counter/*Header pointer.
	for _, item := range d.mountedComponents {
		if item.id == id {
			component = item.comp
			break
		}
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
		root := vr.Render()
		injectComponentID(root, id)
		html = d.renderToHTML(root, &children)
	} else if en, ok := component.(elementNode); ok {
		html = d.renderToHTML(en.AsElement(), &children)
	} else if el, ok := component.(*Element); ok {
		html = d.renderToHTML(el, &children)
	} else {
		html = component.RenderHTML()
	}

	// Replace the element in the DOM
	elRaw := d.document.Call("getElementById", id)
	if elRaw.IsNull() || elRaw.IsUndefined() {
		return fmt.Errf("component element not found: %s", id)
	}
	elRaw.Set("outerHTML", html)

	// Clear element from cache as it was replaced
	d.removeFromElementCache(id)

	// Update lifecycle maps
	d.trackChildren(id, children)

	// Set current component ID for event wiring, OnUpdate and manual listeners
	prevID := d.currentComponentID
	d.currentComponentID = id
	d.wirePendingEvents()

	// Call OnUpdate hook if implemented
	if updatable, ok := component.(Updatable); ok {
		updatable.OnUpdate()
	}

	d.currentComponentID = prevID

	// Mount new children
	for _, child := range children {
		d.mountRecursive(child)
	}

	return nil
}

// Append injects the component's content after the last child of the parent element.
func (d *domWasm) Append(parentID string, component Component) error {
	if component.GetID() == "" {
		component.SetID(generateID())
	}

	var children []Component
	var html string
	if vr, ok := component.(ViewRenderer); ok {
		root := vr.Render()
		injectComponentID(root, component.GetID())
		html = d.renderToHTML(root, &children)
	} else if en, ok := component.(elementNode); ok {
		html = d.renderToHTML(en.AsElement(), &children)
	} else if el, ok := component.(*Element); ok {
		html = d.renderToHTML(el, &children)
	} else {
		html = component.RenderHTML()
	}

	parent := d.getElement(parentID)
	if parent.IsNull() || parent.IsUndefined() {
		return fmt.Errf("parent element not found: %s", parentID)
	}
	parent.Call("insertAdjacentHTML", "beforeend", html)

	d.trackComponent(component)
	d.trackChildren(component.GetID(), children)

	prevID := d.currentComponentID
	d.currentComponentID = component.GetID()
	d.wirePendingEvents()
	d.currentComponentID = prevID

	d.mountRecursive(component)
	for _, child := range children {
		d.mountRecursive(child)
	}
	return nil
}

// unmount removes a component from the DOM and recursively cleans up children.
func (d *domWasm) unmount(component Component) {
	d.unmountRecursive(component)

	// Remove the element from the DOM
	id := component.GetID()
	el := d.document.Call("getElementById", id)
	if !el.IsNull() && !el.IsUndefined() {
		el.Call("remove")
	}

	d.removeFromElementCache(id)
	d.untrackComponent(id)
}

func (d *domWasm) renderToHTML(el *Element, comps *[]Component) string {
	// If the element has events but no ID, generate one
	if len(el.events) > 0 && el.id == "" {
		el.id = generateID()
	}

	for _, ev := range el.events {
		d.pendingEvents = append(d.pendingEvents, struct {
			id      string
			name    string
			handler func(Event)
		}{el.id, ev.Name, ev.Handler})
	}

	s := "<" + el.tag
	if el.id != "" {
		s += " id='" + el.id + "'"
	}
	if len(el.classes) > 0 {
		s += " class='"
		for i, c := range el.classes {
			if i > 0 {
				s += " "
			}
			s += c
		}
		s += "'"
	}
	for _, attr := range el.attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
	s += ">"
	if el.void {
		return s // No children, no closing tag
	}

	for _, child := range el.children {
		switch v := child.(type) {
		case *Element:
			s += d.renderToHTML(v, comps)
		case string:
			s += v
		case elementNode: // NEW: before Component
			*comps = append(*comps, v)
			if v.GetID() == "" {
				v.SetID(generateID())
			}
			s += d.renderToHTML(v.AsElement(), comps)
		case Component:
			*comps = append(*comps, v)
			// Ensure ID
			if v.GetID() == "" {
				v.SetID(generateID())
			}

			if vr, ok := v.(ViewRenderer); ok {
				s += d.renderToHTML(vr.Render(), comps)
			} else if en, ok := v.(elementNode); ok {
				s += d.renderToHTML(en.AsElement(), comps)
			} else if el, ok := v.(*Element); ok {
				s += d.renderToHTML(el, comps)
			} else {
				s += v.RenderHTML()
			}
		default:
			s += fmt.Sprint(v)
		}
	}
	s += "</" + el.tag + ">"
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
	d.untrackComponent(c.GetID())
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
	d.mountedComponents = append(d.mountedComponents, struct {
		id   string
		comp Component
	}{id, c})
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
		d.childrenMap = append(d.childrenMap, struct {
			parentID string
			childIDs []string
		}{parentID, childIDs})
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
			// Track listener for the component that owns the element
			prev := d.currentComponentID
			d.currentComponentID = pe.id
			el.On(pe.name, pe.handler)
			d.currentComponentID = prev
		}
	}
	d.pendingEvents = nil
}

func (d *domWasm) cleanupListeners(id string) {
	var keysToRemove []string
	compIndex := -1
	for i, item := range d.componentListeners {
		if item.id == id {
			keysToRemove = item.keys
			compIndex = i
			break
		}
	}

	if compIndex != -1 {
		d.Log("Cleaning up listeners for ", id, " count: ", len(keysToRemove))
		for _, key := range keysToRemove {
			// Find and remove from eventFuncs
			for i := 0; i < len(d.eventFuncs); i++ {
				ef := d.eventFuncs[i]
				if ef.key == key {
					// Split key into id::type
					// We need the type to call removeEventListener
					parts := d.splitEventKey(key)
					if len(parts) == 2 && !ef.val.IsNull() && !ef.val.IsUndefined() {
						eventType := parts[1]
						ef.val.Call("removeEventListener", eventType, ef.fn)
					}
					ef.fn.Release()

					// Remove from slice
					lastIdx := len(d.eventFuncs) - 1
					d.eventFuncs[i] = d.eventFuncs[lastIdx]
					d.eventFuncs = d.eventFuncs[:lastIdx]
					i-- // Re-check this index as it now contains the last element
				}
			}
		}
		// Remove from componentListeners
		lastIdx := len(d.componentListeners) - 1
		d.componentListeners[compIndex] = d.componentListeners[lastIdx]
		d.componentListeners = d.componentListeners[:lastIdx]
	}
}

func (d *domWasm) splitEventKey(key string) []string {
	// Simple manual split to avoid importing strings just for this
	for i := 0; i < len(key)-1; i++ {
		if key[i] == ':' && key[i+1] == ':' {
			return []string{key[:i], key[i+2:]}
		}
	}
	return nil
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
