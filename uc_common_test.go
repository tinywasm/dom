//go:build wasm

package dom

import (
	"syscall/js"
	"testing"
)

// MockComponent is a simple component for testing.
type MockComponent struct {
	BaseComponent
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
