//go:build wasm

package tinydom

import (
	"syscall/js"
)

// eventWasm is the WASM implementation of the Event interface.
type eventWasm struct {
	js.Value
}

// PreventDefault prevents the default action of the event.
func (e *eventWasm) PreventDefault() {
	e.Call("preventDefault")
}

// StopPropagation stops the event from bubbling up the DOM tree.
func (e *eventWasm) StopPropagation() {
	e.Call("stopPropagation")
}

// TargetValue returns the value of the event's target element.
func (e *eventWasm) TargetValue() string {
	return e.Get("target").Get("value").String()
}
