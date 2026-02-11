//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

// MockComponent is a simple component for testing.
type MockComponent struct {
	*dom.Element
	Mounted bool
}

// HandlerName removed in favor of Identifiable.GetID() provided by BaseComponent

func (c *MockComponent) RenderHTML() string {
	return `<div id="` + c.GetID() + `">Content</div>`
}

func (c *MockComponent) OnMount() {
	c.Mounted = true
}

func (c *MockComponent) OnUnmount() {
	c.Mounted = false
}

func SetupDOM(t *testing.T) js.Value {
	doc := js.Global().Get("document")
	body := doc.Get("body")

	// Do not clear body as it might contain test runner UI

	// Create or get root element
	root := doc.Call("getElementById", "root")
	if root.IsNull() {
		root = doc.Call("createElement", "div")
		root.Set("id", "root")
		body.Call("appendChild", root)
	} else {
		root.Set("innerHTML", "")
	}

	// SetLog(func(v ...any) {
	// 	t.Log(v...)
	// })

	return doc
}

// TestReference is a test-only implementation of dom.Reference for integration tests.
type TestReference struct {
	val js.Value
}

func (r *TestReference) GetAttr(key string) string {
	val := r.val.Call("getAttribute", key)
	if val.IsNull() {
		return ""
	}
	return val.String()
}

func (r *TestReference) Value() string {
	return r.val.Get("value").String()
}

func (r *TestReference) Checked() bool {
	return r.val.Get("checked").Bool()
}

func (r *TestReference) On(eventType string, handler func(event dom.Event)) {
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var e dom.Event
		if len(args) > 0 {
			e = &MockEvent{val: args[0]}
		}
		handler(e)
		return nil
	})
	r.val.Call("addEventListener", eventType, fn)
}

func (r *TestReference) Focus() {
	r.val.Call("focus")
}

// MockEvent implements dom.Event for testing.
type MockEvent struct {
	val js.Value
}

func (e *MockEvent) PreventDefault() {
	e.val.Call("preventDefault")
}

func (e *MockEvent) StopPropagation() {
	e.val.Call("stopPropagation")
}

func (e *MockEvent) TargetID() string {
	target := e.val.Get("target")
	if target.IsNull() || target.IsUndefined() {
		return ""
	}
	return target.Get("id").String()
}

func (e *MockEvent) TargetValue() string {
	target := e.val.Get("target")
	if target.IsNull() || target.IsUndefined() {
		return ""
	}
	return target.Get("value").String()
}

func (e *MockEvent) TargetChecked() bool {
	target := e.val.Get("target")
	if target.IsNull() || target.IsUndefined() {
		return false
	}
	return target.Get("checked").Bool()
}

// GetRef is a test helper to get a Reference for an element by ID.
func GetRef(id string) (dom.Reference, bool) {
	var val js.Value
	doc := js.Global().Get("document")
	switch id {
	case "body":
		val = doc.Get("body")
	case "head":
		val = doc.Get("head")
	default:
		val = doc.Call("getElementById", id)
	}

	if val.IsNull() || val.IsUndefined() {
		return nil, false
	}
	return &TestReference{val: val}, true
}
