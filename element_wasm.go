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
func (e *elementWasm) Focus() {
	e.val.Call("focus")
}
