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
	localStorage js.Value // Cached localStorage object
	lsUsedBytes  int      // Current localStorage budget usage in bytes (UTF-16)

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
		ownerID string
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
	initedIDs []string
	cleanups  []struct {
		id string
		fn func()
	}
	unsubs []struct {
		id    string
		unsub func()
	}
	updating []string
}

// newDom returns a new instance of the domWasm.
func newDom(td *tinyDOM) DOM {
	ls := js.Global().Get("localStorage")
	used := 0
	if ls.Truthy() {
		// Initial O(n) scan — occurs once at startup to initialize budget counter.
		length := ls.Get("length").Int()
		for i := 0; i < length; i++ {
			key := ls.Call("key", i).String()
			if val := ls.Call("getItem", key); !val.IsNull() && !val.IsUndefined() {
				used += lsEntrySize(key, val.String())
			}
		}
	}
	return &domWasm{
		tinyDOM:      td,
		document:     js.Global().Get("document"),
		localStorage: ls,
		lsUsedBytes:  used,
	}
}

// lsEntrySize estimates the UTF-16 byte size of a localStorage entry.
func lsEntrySize(key, value string) int { return (len(key) + len(value)) * 2 }

// Get retrieves an element by ID from the cache or the DOM.
func (d *domWasm) Get(id string) (Reference, bool) {
	// Linear search in cache
	for i, item := range d.elementCache {
		if item.id == id {
			// Invalidate stale cache if element was removed/replaced
			if !item.val.Get("isConnected").Bool() {
				lastIdx := len(d.elementCache) - 1
				d.elementCache[i] = d.elementCache[lastIdx]
				d.elementCache = d.elementCache[:lastIdx]
				break
			}
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

type domCtx struct {
	id string
	d  *domWasm
}

func (c *domCtx) OnCleanup(fn func()) {
	c.d.cleanups = append(c.d.cleanups, struct {
		id string
		fn func()
	}{c.id, fn})
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

	d.initComponent(component)

	// Render to HTML and collect child components
	var children []Component
	var html string

	if vr, ok := component.(ViewRenderer); ok {
		root := vr.Render()
		injectComponentID(root, component.GetID())
		html = d.renderToHTML(root, &children, component.GetID())
	} else if en, ok := component.(elementNode); ok {
		html = d.renderToHTML(en.AsElement(), &children, component.GetID())
	} else if el, ok := component.(*Element); ok {
		html = d.renderToHTML(el, &children, component.GetID())
	} else {
		html = component.String()
	}

	parent := d.getElement(parentID)
	if parent.IsNull() || parent.IsUndefined() {
		return fmt.Errf("parent element not found: %s", parentID)
	}

	// Clean up any existing components in this parent before wiping content.
	d.cleanupChildren(parentID)

	parent.Set("innerHTML", html)

	// Update lifecycle maps
	d.trackComponent(component)
	d.trackChildren(component.GetID(), children)

	// Set current component ID for event wiring
	prevID := d.currentComponentID
	d.currentComponentID = component.GetID()
	d.wirePendingEvents()
	d.wireBindings(component.GetID())
	d.currentComponentID = prevID

	for _, child := range children {
		d.mountRecursive(child)
	}

	return nil
}

func (d *domWasm) initComponent(c Component) {
	if c == nil {
		return
	}
	id := c.GetID()
	for _, initedID := range d.initedIDs {
		if initedID == id {
			return
		}
	}

	if initable, ok := c.(initable); ok {
		initable.Init(&domCtx{id: id, d: d})
	}
	d.initedIDs = append(d.initedIDs, id)
}

// update re-renders the component and replaces it in the DOM.
func (d *domWasm) update(id string) {
	if d.document.IsNull() || d.document.IsUndefined() {
		d.Log("tinywasm/dom: document not found in update")
		return
	}

	for _, uid := range d.updating {
		if uid == id {
			if d.devMode {
				d.Log("tinywasm/dom: re-entrant update on", id, "ignored")
			}
			return
		}
	}
	d.updating = append(d.updating, id)

	var component Component
	// Resolve the full outer component from tracked references.
	for _, item := range d.mountedComponents {
		if item.id == id {
			component = item.comp
			break
		}
	}

	if component == nil {
		// Remove from updating before returning
		for i, uid := range d.updating {
			if uid == id {
				d.updating = append(d.updating[:i], d.updating[i+1:]...)
				break
			}
		}
		return
	}

	// Clean up old children listeners/lifecycle
	d.cleanupChildren(id)
	d.cleanupListeners(id)

	var children []Component
	var html string

	if vr, ok := component.(ViewRenderer); ok {
		root := vr.Render()
		injectComponentID(root, id)
		html = d.renderToHTML(root, &children, id)
	} else if en, ok := component.(elementNode); ok {
		html = d.renderToHTML(en.AsElement(), &children, id)
	} else if el, ok := component.(*Element); ok {
		html = d.renderToHTML(el, &children, id)
	} else {
		html = component.String()
	}

	// Replace the element in the DOM
	elRaw := d.document.Call("getElementById", id)
	if elRaw.IsNull() || elRaw.IsUndefined() {
		if d.devMode {
			d.Log("tinywasm/dom: component element not found during update:", id, "(this usually means the component root element has no ID)")
		}
		// Remove from updating before returning
		for i, uid := range d.updating {
			if uid == id {
				d.updating = append(d.updating[:i], d.updating[i+1:]...)
				break
			}
		}
		return
	}

	// Snapshot active element and cursor before outerHTML destroys them.
	activeEl := d.document.Get("activeElement")
	activeID := ""
	cursorStart, cursorEnd := 0, 0
	if !activeEl.IsNull() && !activeEl.IsUndefined() {
		activeID = activeEl.Get("id").String()
		cs := activeEl.Get("selectionStart")
		ce := activeEl.Get("selectionEnd")
		if !cs.IsNull() && !cs.IsUndefined() {
			cursorStart = cs.Int()
		}
		if !ce.IsNull() && !ce.IsUndefined() {
			cursorEnd = ce.Int()
		}
	}

	elRaw.Set("outerHTML", html)

	// Clear element from cache as it was replaced
	d.removeFromElementCache(id)

	// Update lifecycle maps
	d.trackChildren(id, children)

	// Set current component ID for event wiring
	prevID := d.currentComponentID
	d.currentComponentID = id
	d.wirePendingEvents()
	d.wireBindings(id)
	d.currentComponentID = prevID

	// Mount new children
	for _, child := range children {
		d.mountRecursive(child)
	}

	// Restore focus and cursor to the element that was active before outerHTML replacement.
	if activeID != "" {
		restored := d.document.Call("getElementById", activeID)
		if !restored.IsNull() && !restored.IsUndefined() {
			currentActive := d.document.Get("activeElement")
			alreadyActive := !currentActive.IsNull() && !currentActive.IsUndefined() &&
				currentActive.Get("id").String() == activeID
			if !alreadyActive {
				restored.Call("focus")
			}
			cs := restored.Get("selectionStart")
			if !cs.IsNull() && !cs.IsUndefined() {
				restored.Call("setSelectionRange", cursorStart, cursorEnd)
			}
		}
	}

	for i, uid := range d.updating {
		if uid == id {
			d.updating = append(d.updating[:i], d.updating[i+1:]...)
			break
		}
	}
}

// Update re-renders the component (NOT public anymore, but needed for internal reasons? No, PLAN says unexport)
// ActuallyPLAN says: "Update re-renders a component." -> was dom.Update(comp).
// PLAN Change 4 says: "Unexport Update -> update (an internal primitive used by Show/BindChildren...)"

// Append injects the component's content after the last child of the parent element.
func (d *domWasm) Append(parentID string, component Component) error {
	if component.GetID() == "" {
		component.SetID(generateID())
	}

	d.initComponent(component)

	var children []Component
	var html string
	if vr, ok := component.(ViewRenderer); ok {
		root := vr.Render()
		injectComponentID(root, component.GetID())
		html = d.renderToHTML(root, &children, component.GetID())
	} else if en, ok := component.(elementNode); ok {
		html = d.renderToHTML(en.AsElement(), &children, component.GetID())
	} else if el, ok := component.(*Element); ok {
		html = d.renderToHTML(el, &children, component.GetID())
	} else {
		html = component.String()
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
	d.wireBindings(component.GetID())
	d.currentComponentID = prevID

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

func (d *domWasm) renderToHTML(el *Element, comps *[]Component, ownerID string) string {
	if el == nil {
		if d.devMode {
			d.Log("tinywasm/dom: nil Element encountered during renderToHTML (pointer-embedded Element mistake?)")
		}
		return ""
	}
	// If the element has events or bindings but no ID, generate one
	if (len(el.events) > 0 || len(el.bindings) > 0 || el.autofocus) && el.id == "" {
		el.id = generateID()
	}

	for _, ev := range el.events {
		d.pendingEvents = append(d.pendingEvents, struct {
			id      string
			ownerID string
			name    string
			handler func(Event)
		}{el.id, ownerID, ev.Name, ev.Handler})
	}

	s := "<" + el.tag
	if el.id != "" {
		s += " id='" + el.id + "'"
	}

	// Apply bindings initial state
	classes := el.classes
	attrs := el.attrs
	textContent := ""
	hasTextContent := false

	for _, b := range el.bindings {
		switch b.kind {
		case "text":
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalString); ok {
					textContent = sig.Get()
				}
			} else if b.fnString != nil {
				textContent = b.fnString()
			}
			hasTextContent = true
		case "attr":
			val := ""
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalString); ok {
					val = sig.Get()
				}
			} else if b.fnString != nil {
				val = b.fnString()
			}
			// Replace existing attr if found
			found := false
			for i, attr := range attrs {
				if attr.Key == b.name {
					attrs[i].Value = val
					found = true
					break
				}
			}
			if !found {
				attrs = append(attrs, fmt.KeyValue{Key: b.name, Value: val})
			}
		case "class":
			on := false
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalBool); ok {
					on = sig.Get()
				}
			} else if b.fnBool != nil {
				on = b.fnBool()
			}
			if on {
				classes = append(classes, b.name)
			}
		case "attrbool":
			on := false
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalBool); ok {
					on = sig.Get()
				}
			} else if b.fnBool != nil {
				on = b.fnBool()
			}
			if on {
				attrs = append(attrs, fmt.KeyValue{Key: b.name, Value: ""})
			}
		case "value":
			val := ""
			if b.signal != nil {
				if sig, ok := b.signal.(*SignalString); ok {
					val = sig.Get()
				}
			}
			attrs = append(attrs, fmt.KeyValue{Key: "value", Value: val})
		}
	}

	if len(classes) > 0 {
		s += " class='"
		for i, c := range classes {
			if i > 0 {
				s += " "
			}
			s += c
		}
		s += "'"
	}
	for _, attr := range attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
	s += ">"
	if el.void {
		return s // No children, no closing tag
	}

	if hasTextContent {
		s += textContent
	} else {
		for _, child := range el.children {
			switch v := child.(type) {
			case *Element:
				s += d.renderToHTML(v, comps, ownerID)
			case string:
				s += v
			case Component:
				if v == nil {
					if d.devMode {
						d.Log("tinywasm/dom: nil Component encountered (pointer-embedded Element mistake?)")
					}
					continue
				}
				*comps = append(*comps, v)
				if v.GetID() == "" {
					v.SetID(generateID())
				}
				childID := v.GetID()
				d.initComponent(v)

				if vr, ok := v.(ViewRenderer); ok {
					root := vr.Render()
					injectComponentID(root, childID)
					s += d.renderToHTML(root, comps, childID)
				} else if en, ok := v.(elementNode); ok {
					root := en.AsElement()
					injectComponentID(root, childID)
					s += d.renderToHTML(root, comps, childID)
				} else if el, ok := v.(*Element); ok {
					injectComponentID(el, childID)
					s += d.renderToHTML(el, comps, childID)
				} else {
					s += v.String()
				}
			default:
				s += fmt.Sprint(v)
			}
		}
	}
	s += "</" + el.tag + ">"
	return s
}

func (d *domWasm) mountRecursive(c Component) {
	if c == nil {
		return
	}
	prevID := d.currentComponentID
	d.currentComponentID = c.GetID()
	defer func() { d.currentComponentID = prevID }()

	// Mount children (for HTMLRenderer primarily)
	for _, child := range c.Children() {
		if child != nil {
			d.mountRecursive(child)
		}
	}

	// Children tracked in maps are mounted in Render/Update loop
}

func (d *domWasm) unmountRecursive(c Component) {
	if c == nil {
		return
	}
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
			d.cleanupSignalSubscriptions(childID)
			d.runCleanups(childID)
		}
	}

	d.cleanupListeners(c.GetID())
	d.cleanupSignalSubscriptions(c.GetID())
	d.runCleanups(c.GetID())
	d.untrackComponent(c.GetID())

	// Remove from initedIDs so it can be re-inited if re-mounted
	for i, initedID := range d.initedIDs {
		if initedID == c.GetID() {
			d.initedIDs = append(d.initedIDs[:i], d.initedIDs[i+1:]...)
			break
		}
	}
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
	var childIDs []string
	for _, c := range children {
		if c == nil {
			continue
		}
		childIDs = append(childIDs, c.GetID())
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
			d.currentComponentID = pe.ownerID
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
		for _, key := range keysToRemove {
			// Find and remove from eventFuncs
			for i := 0; i < len(d.eventFuncs); i++ {
				ef := d.eventFuncs[i]
				if ef.key == key {
					// Split key into id::type
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
					i--
				}
			}
		}
		// Remove from componentListeners
		lastIdx := len(d.componentListeners) - 1
		d.componentListeners[compIndex] = d.componentListeners[lastIdx]
		d.componentListeners = d.componentListeners[:lastIdx]
	}
}

func (d *domWasm) wireBindings(id string) {
	// Find component and all its elements with bindings
	var component Component
	for _, item := range d.mountedComponents {
		if item.id == id {
			component = item.comp
			break
		}
	}
	if component == nil {
		return
	}

	if vr, ok := component.(ViewRenderer); ok {
		d.wireElementBindings(vr.Render(), id)
	} else if en, ok := component.(elementNode); ok {
		d.wireElementBindings(en.AsElement(), id)
	} else if el, ok := component.(*Element); ok {
		d.wireElementBindings(el, id)
	}
}

func (d *domWasm) wireElementBindings(el *Element, ownerID string) {
	if el == nil {
		return
	}
	if el.autofocus {
		if ref, ok := d.Get(el.id); ok {
			// Focus iff nothing else is focused
			activeEl := d.document.Get("activeElement")
			if activeEl.IsNull() || activeEl.IsUndefined() || activeEl.Get("tagName").String() == "BODY" {
				ref.Focus()
			}
		}
	}

	for _, b := range el.bindings {
		b := b
		ref, ok := d.Get(el.id)
		if !ok {
			continue
		}

		var updater func()
		switch b.kind {
		case "text":
			updater = func() {
				val := ""
				if b.signal != nil {
					if sig, ok := b.signal.(*SignalString); ok {
						val = sig.Get()
					}
				} else if b.fnString != nil {
					val = b.fnString()
				}
				ref.SetText(val)
				if d.devMode {
					d.Log("[dom] patch #"+el.id+" textContent:", val)
				}
			}
		case "attr":
			updater = func() {
				val := ""
				if b.signal != nil {
					if sig, ok := b.signal.(*SignalString); ok {
						val = sig.Get()
					}
				} else if b.fnString != nil {
					val = b.fnString()
				}
				ref.SetAttr(b.name, val)
				if d.devMode {
					d.Log("[dom] patch #"+el.id+" attr "+b.name+":", val)
				}
			}
		case "class":
			updater = func() {
				on := false
				if b.signal != nil {
					if sig, ok := b.signal.(*SignalBool); ok {
						on = sig.Get()
					}
				} else if b.fnBool != nil {
					on = b.fnBool()
				}
				if on {
					ref.(*elementWasm).val.Get("classList").Call("add", b.name)
				} else {
					ref.(*elementWasm).val.Get("classList").Call("remove", b.name)
				}
				if d.devMode {
					d.Log("[dom] patch #"+el.id+" class "+b.name+":", on)
				}
			}
		case "attrbool":
			updater = func() {
				on := false
				if b.signal != nil {
					if sig, ok := b.signal.(*SignalBool); ok {
						on = sig.Get()
					}
				} else if b.fnBool != nil {
					on = b.fnBool()
				}
				if on {
					ref.SetAttr(b.name, "")
				} else {
					ref.RemoveAttr(b.name)
				}
				if d.devMode {
					d.Log("[dom] patch #"+el.id+" attrbool "+b.name+":", on)
				}
			}
		case "value":
			// Two-way binding
			if b.signal == nil {
				continue
			}
			sig, ok := b.signal.(*SignalString)
			if !ok {
				continue
			}

			// Check if element is input/textarea
			tagName := ref.(*elementWasm).val.Get("tagName").String()
			if tagName != "INPUT" && tagName != "TEXTAREA" {
				if d.devMode {
					d.Log("tinywasm/dom: Bind used on non-input element:", tagName)
				}
			}

			updater = func() {
				val := sig.Get()
				// Skip if activeElement to avoid cursor jumps
				activeEl := d.document.Get("activeElement")
				if !activeEl.IsNull() && !activeEl.IsUndefined() && activeEl.Get("id").String() == el.id {
					return
				}
				if ref.Value() != val {
					ref.SetValue(val)
				}
			}

			// Listen for input changes
			ref.On("input", func(e Event) {
				sig.Set(ref.Value())
			})
		case "children":
			sig, ok := b.signal.(*SignalNodes)
			if !ok {
				continue
			}
			updater = func() {
				d.reconcileChildren(el.id, sig.Get())
			}
		}

		if updater != nil {
			if b.signal != nil {
				unsub := b.signal.subscribe(updater)
				d.unsubs = append(d.unsubs, struct {
					id    string
					unsub func()
				}{ownerID, unsub})
			} else if b.fnString != nil || b.fnBool != nil {
				// Use tracker for computed bindings
				t := &tracker{}
				prev := currentTracker
				currentTracker = t
				updater() // Initial run to track
				currentTracker = prev

				for _, s := range t.signals {
					unsub := s.subscribe(updater)
					d.unsubs = append(d.unsubs, struct {
						id    string
						unsub func()
					}{ownerID, unsub})
				}
			}
		}
	}

	for _, child := range el.children {
		if childEl, ok := child.(*Element); ok {
			d.wireElementBindings(childEl, ownerID)
		}
	}
}

func (d *domWasm) cleanupSignalSubscriptions(id string) {
	for i := 0; i < len(d.unsubs); i++ {
		if d.unsubs[i].id == id {
			d.unsubs[i].unsub()
			d.unsubs = append(d.unsubs[:i], d.unsubs[i+1:]...)
			i--
		}
	}
}

func (d *domWasm) runCleanups(id string) {
	for i := 0; i < len(d.cleanups); i++ {
		if d.cleanups[i].id == id {
			d.cleanups[i].fn()
			d.cleanups = append(d.cleanups[:i], d.cleanups[i+1:]...)
			i--
		}
	}
}

func (d *domWasm) reconcileChildren(parentID string, newNodes []*Element) {
	parent, ok := d.Get(parentID)
	if !ok {
		return
	}
	parentVal := parent.(*elementWasm).val

	// Keyed reconcile
	existingNodes := parentVal.Get("children")
	existingLen := existingNodes.Get("length").Int()

	// Build map of current children by key
	currentKeys := make([]struct {
		key string
		val js.Value
	}, existingLen)
	for i := 0; i < existingLen; i++ {
		node := existingNodes.Call("item", i)
		currentKeys[i] = struct {
			key string
			val js.Value
		}{node.Get("id").String(), node}
	}

	// Dev mode key validation
	if d.devMode {
		keys := make([]string, 0, len(newNodes))
		for _, n := range newNodes {
			if n.key == "" && n.id == "" {
				d.Log("tinywasm/dom: row in BindChildren has no key/id (volatile identity)")
			}
			key := n.key
			if key == "" {
				key = n.id
			}
			for _, existingKey := range keys {
				if key != "" && existingKey == key {
					d.Log("tinywasm/dom: duplicate key in BindChildren:", key)
				}
			}
			keys = append(keys, key)
		}
	}

	// Simplistic reconciliation: for each new node, if it exists, move it; otherwise insert it.
	// Then remove any old nodes that aren't in the new set.
	var comps []Component
	for i, n := range newNodes {
		key := n.key
		if key == "" {
			key = n.id
		}
		if key == "" {
			key = generateID()
			n.id = key
		}

		var found js.Value
		for _, item := range currentKeys {
			if item.key == key {
				found = item.val
				break
			}
		}

		if !found.IsUndefined() && !found.IsNull() {
			// Move to correct position if needed
			if i < existingLen && !existingNodes.Call("item", i).Equal(found) {
				parentVal.Call("insertBefore", found, existingNodes.Call("item", i))
			}
		} else {
			// Create and insert
			html := d.renderToHTML(n, &comps, parentID)
			tempDiv := d.document.Call("createElement", "div")
			tempDiv.Set("innerHTML", html)
			newNode := tempDiv.Get("firstElementChild")
			if i < existingLen {
				parentVal.Call("insertBefore", newNode, existingNodes.Call("item", i))
			} else {
				parentVal.Call("appendChild", newNode)
			}
			// Wire bindings and events for the new node only
			d.wireElementBindings(n, parentID)
		}
	}

	// Remove extra nodes
	for existingLen > len(newNodes) {
		last := parentVal.Get("lastElementChild")
		lastID := last.Get("id").String()
		last.Call("remove")
		d.cleanupListeners(lastID)
		d.cleanupSignalSubscriptions(lastID)
		d.runCleanups(lastID)
		existingLen--
	}

	d.wirePendingEvents()
	for _, c := range comps {
		d.mountRecursive(c)
	}
}

// Show mounts/unmounts a subtree when cond flips. Runs the rendered subtree's Init/cleanup.
func Show(cond *SignalBool, render func() *Element) *Element {
	containerID := generateID()
	container := NewElement("div").ID(containerID)

	var lastSubtreeID string

	updater := func() {
		if ref, ok := instance.Get(containerID); ok {
			// Cleanup previous subtree
			if lastSubtreeID != "" {
				instance.(*domWasm).cleanupListeners(lastSubtreeID)
				instance.(*domWasm).cleanupSignalSubscriptions(lastSubtreeID)
				instance.(*domWasm).runCleanups(lastSubtreeID)
				lastSubtreeID = ""
			}
			instance.(*domWasm).cleanupChildren(containerID)

			ref.(*elementWasm).val.Set("innerHTML", "")
			if cond.Get() {
				root := render()
				if root.id == "" {
					root.id = generateID()
				}
				lastSubtreeID = root.id

				var comps []Component
				html := instance.(*domWasm).renderToHTML(root, &comps, containerID)
				ref.(*elementWasm).val.Set("innerHTML", html)
				instance.(*domWasm).wirePendingEvents()
				instance.(*domWasm).wireElementBindings(root, containerID)
				for _, c := range comps {
					instance.(*domWasm).mountRecursive(c)
				}
			}
		}
	}

	unsub := cond.subscribe(updater)

	// Initial state
	if cond.Get() {
		root := render()
		container.children = append(container.children, root)
		if root.id == "" {
			root.id = generateID()
		}
		lastSubtreeID = root.id
	}

	// Register unsub to be called when container is unmounted
	instance.(*domWasm).unsubs = append(instance.(*domWasm).unsubs, struct {
		id    string
		unsub func()
	}{containerID, unsub})

	return container
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
