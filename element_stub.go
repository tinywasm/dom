//go:build !wasm

package tinydom

// elementStub is a no-op implementation of the Element interface for non-WASM targets.
type elementStub struct{}

// SetText does nothing.
func (e *elementStub) SetText(text string) {}

// SetHTML does nothing.
func (e *elementStub) SetHTML(html string) {}

// AppendHTML does nothing.
func (e *elementStub) AppendHTML(html string) {}

// Remove does nothing.
func (e *elementStub) Remove() {}

// AddClass does nothing.
func (e *elementStub) AddClass(class string) {}

// RemoveClass does nothing.
func (e *elementStub) RemoveClass(class string) {}

// ToggleClass does nothing.
func (e *elementStub) ToggleClass(class string) {}

// SetAttr does nothing.
func (e *elementStub) SetAttr(key, value string) {}

// GetAttr returns an empty string.
func (e *elementStub) GetAttr(key string) string { return "" }

// RemoveAttr does nothing.
func (e *elementStub) RemoveAttr(key string) {}

// Value returns an empty string.
func (e *elementStub) Value() string { return "" }

// SetValue does nothing.
func (e *elementStub) SetValue(value string) {}

// Click does nothing.
func (e *elementStub) Click(handler func(event Event)) {}

// On does nothing.
func (e *elementStub) On(eventType string, handler func(event Event)) {}

// Focus does nothing.
func (e *elementStub) Focus() {}
