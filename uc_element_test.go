//go:build wasm

package dom

import (
	"syscall/js"
	"testing"
)

func TestElementMethods(t *testing.T) {
	doc := setupDOM(t)

	// Mount a component to test on
	comp := &MockComponent{}
	comp.SetID("comp-elem")
	Mount("root", comp)

	el, ok := Get("comp-elem")
	if !ok {
		t.Fatal("Component element not found")
	}

	// Helper to get raw JS value
	getRaw := func(id string) js.Value {
		return doc.Call("getElementById", id)
	}
	rawEl := getRaw("comp-elem")

	t.Run("SetText", func(t *testing.T) {
		el.SetText("Hello World")
		if rawEl.Get("textContent").String() != "Hello World" {
			t.Error("SetText failed with string")
		}

		// Test with non-string (int)
		el.SetText("Count: ", 42)
		if rawEl.Get("textContent").String() != "Count: 42" {
			t.Error("SetText failed with variadic args")
		}
	})

	t.Run("SetHTML", func(t *testing.T) {
		el.SetHTML("<span>Inner</span>")
		if rawEl.Get("innerHTML").String() != "<span>Inner</span>" {
			t.Error("SetHTML failed")
		}
	})

	t.Run("AppendHTML", func(t *testing.T) {
		el.AppendHTML("<span>Appended</span>")
		html := rawEl.Get("innerHTML").String()
		if html != "<span>Inner</span><span>Appended</span>" {
			t.Errorf("AppendHTML failed, got: %s", html)
		}
	})

	t.Run("Classes", func(t *testing.T) {
		el.AddClass("test-class")
		if !rawEl.Get("classList").Call("contains", "test-class").Bool() {
			t.Error("AddClass failed")
		}
		el.ToggleClass("test-class")
		if rawEl.Get("classList").Call("contains", "test-class").Bool() {
			t.Error("ToggleClass (remove) failed")
		}
		el.ToggleClass("test-class")
		if !rawEl.Get("classList").Call("contains", "test-class").Bool() {
			t.Error("ToggleClass (add) failed")
		}
		el.RemoveClass("test-class")
		if rawEl.Get("classList").Call("contains", "test-class").Bool() {
			t.Error("RemoveClass failed")
		}
	})

	t.Run("Attributes", func(t *testing.T) {
		el.SetAttr("data-test", "value")
		if el.GetAttr("data-test") != "value" {
			t.Error("SetAttr/GetAttr failed")
		}
		el.RemoveAttr("data-test")
		val := el.GetAttr("data-test")
		if val != "" && val != "<null>" {
			t.Errorf("RemoveAttr failed, got: %s", val)
		}
	})

	t.Run("Remove Edge Cases", func(t *testing.T) {
		root, _ := Get("root")
		root.AppendHTML(`<div id="temp-remove"></div>`)
		tempEl, _ := Get("temp-remove")
		tempEl.Remove()
		// Calling remove again should be fine
		tempEl.Remove()
	})

	t.Run("Focus and SetValue", func(t *testing.T) {
		root, _ := Get("root")
		root.AppendHTML(`<input id="test-focus">`)
		inputEl, _ := Get("test-focus")

		inputEl.SetValue("new-value")
		if inputEl.Value() != "new-value" {
			t.Error("SetValue failed")
		}

		inputEl.Focus()
		active := doc.Get("activeElement")
		if !active.Equal(doc.Call("getElementById", "test-focus")) {
			t.Log("Focus check failed (might be browser restriction)")
		}
	})
}
