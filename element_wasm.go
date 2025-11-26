//go:build wasm

package tinydom

import (
	"syscall/js"
)

// elementWasm is the WASM implementation of the Element interface.
type elementWasm struct {
	js.Value
	dom *domWasm
	id  string
}

// SetText sets the text content of the element.
func (e *elementWasm) SetText(text string) {
	e.Set("textContent", text)
}

// SetHTML sets the inner HTML of the element.
func (e *elementWasm) SetHTML(html string) {
	e.Set("innerHTML", html)
}

// AppendHTML adds HTML to the end of the element's content.
func (e *elementWasm) AppendHTML(html string) {
	e.Call("insertAdjacentHTML", "beforeend", html)
}

// Remove removes the element from the DOM.
func (e *elementWasm) Remove() {
	if e.Value.IsNull() || e.Value.IsUndefined() {
		return
	}
	e.Call("remove")
}

// AddClass adds a CSS class to the element.
func (e *elementWasm) AddClass(class string) {
	e.Get("classList").Call("add", class)
}

// RemoveClass removes a CSS class from the element.
func (e *elementWasm) RemoveClass(class string) {
	e.Get("classList").Call("remove", class)
}

// ToggleClass toggles a CSS class.
func (e *elementWasm) ToggleClass(class string) {
	e.Get("classList").Call("toggle", class)
}

// SetAttr sets an attribute value.
func (e *elementWasm) SetAttr(key, value string) {
	e.Call("setAttribute", key, value)
}

// GetAttr retrieves an attribute value.
func (e *elementWasm) GetAttr(key string) string {
	return e.Call("getAttribute", key).String()
}

// RemoveAttr removes an attribute.
func (e *elementWasm) RemoveAttr(key string) {
	e.Call("removeAttribute", key)
}

// Value returns the current value of an input/textarea/select.
func (e *elementWasm) Value() string {
	return e.Get("value").String()
}

// SetValue sets the value of an input/textarea/select.
func (e *elementWasm) SetValue(value string) {
	e.Set("value", value)
}

// Click registers a click event handler.
func (e *elementWasm) Click(handler func(event Event)) {
	e.On("click", handler)
}

// On registers a generic event handler.
func (e *elementWasm) On(eventType string, handler func(event Event)) {
	eventKey := e.id + "::" + eventType
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		evt := eventWasm{Value: args[0]}
		handler(&evt)
		return nil
	})
	e.Call("addEventListener", eventType, fn)
	e.dom.eventFuncs[eventKey] = fn

	// Associate the event with the component currently being mounted.
	if e.dom.currentComponentID != "" {
		compID := e.dom.currentComponentID
		e.dom.componentListeners[compID] = append(e.dom.componentListeners[compID], eventKey)
	}
}

// Focus sets focus to the element.
func (e *elementWasm) Focus() {
	e.Call("focus")
}
