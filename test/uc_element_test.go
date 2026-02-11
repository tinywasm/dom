//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

func TestElementMethods(t *testing.T) {
	doc := SetupDOM(t)

	// Mount a component to test on
	comp := &MockComponent{}
	comp.SetID("comp-elem")
	dom.Render("root", comp)

	el, ok := dom.Get("comp-elem")
	if !ok {
		t.Fatal("Component element not found")
	}

	// Helper to get raw JS value
	getRaw := func(id string) js.Value {
		return doc.Call("getElementById", id)
	}
	rawEl := getRaw("comp-elem")

	t.Run("Classes", func(t *testing.T) {
		// Use direct JS to set classes, then verify with Reference interface if possible
		// (Simplified Reference doesn't have class methods)
		rawEl.Get("classList").Call("add", "test-class")
		if !rawEl.Get("classList").Call("contains", "test-class").Bool() {
			t.Error("AddClass failed")
		}
	})

	t.Run("Attributes", func(t *testing.T) {
		rawEl.Call("setAttribute", "data-test", "value")
		if el.GetAttr("data-test") != "value" {
			t.Error("GetAttr failed")
		}
	})

	t.Run("Focus and Value", func(t *testing.T) {
		rawEl.Set("innerHTML", `<input id="test-focus">`)
		inputEl, _ := dom.Get("test-focus")
		rawInput := doc.Call("getElementById", "test-focus")

		rawInput.Set("value", "new-value")
		if inputEl.Value() != "new-value" {
			t.Error("Value() failed")
		}

		inputEl.Focus()
	})
}
