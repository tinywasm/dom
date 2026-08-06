//go:build wasm

package dom

import (
	"syscall/js"
)

// elementWasm is the WASM implementation of the Reference interface.
type elementWasm struct {
	val js.Value
	dom *domWasm
	id  string
}

// GetAttr retrieves an attribute value.
func (e *elementWasm) GetAttr(key string) string {
	return e.val.Call("getAttribute", key).String()
}

// Value returns the current value of an input/textarea/select.
func (e *elementWasm) Value() string {
	return e.val.Get("value").String()
}

// SetValue sets element.value.
func (e *elementWasm) SetValue(value string) {
	e.val.Set("value", value)
}

// SetAttr calls element.setAttribute.
func (e *elementWasm) SetAttr(key, value string) {
	e.val.Call("setAttribute", key, value)
}

// RemoveAttr calls element.removeAttribute.
func (e *elementWasm) RemoveAttr(key string) {
	e.val.Call("removeAttribute", key)
}

// SetText sets element.textContent.
func (e *elementWasm) SetText(text string) {
	e.val.Set("textContent", text)
}

// Checked returns current checked state.
func (e *elementWasm) Checked() bool {
	return e.val.Get("checked").Bool()
}

// On registers a generic event handler.
func (e *elementWasm) On(eventType string, handler func(event Event)) {
	eventKey := e.id + "::" + eventType
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		evt := eventWasm{Value: args[0]}
		handler(&evt)
		return nil
	})
	e.val.Call("addEventListener", eventType, fn)

	// Append to eventFuncs
	e.dom.eventFuncs = append(e.dom.eventFuncs, struct {
		key string
		val js.Value
		fn  js.Func
	}{eventKey, e.val, fn})

	// Associate the event with the component currently being mounted.
	if e.dom.currentComponentID != "" {
		compID := e.dom.currentComponentID
		found := false
		for i, item := range e.dom.componentListeners {
			if item.id == compID {
				e.dom.componentListeners[i].keys = append(e.dom.componentListeners[i].keys, eventKey)
				found = true
				break
			}
		}
		if !found {
			e.dom.componentListeners = append(e.dom.componentListeners, struct {
				id   string
				keys []string
			}{compID, []string{eventKey}})
		}
	}
}

// Focus sets focus to the element.
//
// preventScroll is part of the contract, not a tweak: focus() otherwise makes
// the browser scroll EVERY scrollable ancestor to reveal the element, and here
// the layout owns scroll position, not the browser — ScrollIntoView below is
// the explicit way to move it. The two fight. On iOS Safari the browser's
// version wins and jumps a scroll-snap strip instantly, which both kills the
// smooth scroll the layout asked for and, once the keyboard is up, can leave
// the strip parked where it was with the caret on an off-screen field.
//
// Nothing depends on the suppressed behavior: every Focus() caller either
// targets an element that is already on screen (autofocus at mount, a search
// input inside a dropdown that just opened) or scrolls its own container
// itself. Keyboard opening is unaffected — that follows from focus() being
// synchronous with the user gesture, not from the scroll. Browsers without
// preventScroll ignore the options object and behave exactly as before.
func (e *elementWasm) Focus() {
	opts := js.Global().Get("Object").New()
	opts.Set("preventScroll", true)
	e.val.Call("focus", opts)
}

// ScrollsX reports whether the element's content overflows its box along the
// inline axis. The 1px slack absorbs sub-pixel layout rounding, which otherwise
// reports a strip as scrollable when it is exactly full.
func (e *elementWasm) ScrollsX() bool {
	return e.val.Get("scrollWidth").Float() > e.val.Get("clientWidth").Float()+1
}

// ScrollIntoView smooth-scrolls the element into view.
func (e *elementWasm) ScrollIntoView() {
	opts := js.Global().Get("Object").New()
	opts.Set("behavior", "smooth")
	opts.Set("inline", "start")
	opts.Set("block", "nearest")
	e.val.Call("scrollIntoView", opts)
}
