//go:build !wasm

package tinydom

// eventStub is a no-op implementation of the Event interface for non-WASM targets.
type eventStub struct{}

// PreventDefault does nothing.
func (e *eventStub) PreventDefault() {}

// StopPropagation does nothing.
func (e *eventStub) StopPropagation() {}

// TargetValue returns an empty string.
func (e *eventStub) TargetValue() string { return "" }
