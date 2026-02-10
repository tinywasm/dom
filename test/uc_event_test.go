//go:build wasm

package dom_test

import (
	"github.com/tinywasm/dom"
	"syscall/js"
	"testing"
)

// EventComponent registers listeners in OnMount
type EventComponent struct {
	MockComponent
	clickCount  int
	customCount int
}

func (c *EventComponent) OnMount() {
	c.MockComponent.OnMount()
	// Register events using the global API
	el, ok := dom.Get(c.GetID())
	if ok {
		el.On("click", func(e dom.Event) {
			c.clickCount++
			e.PreventDefault()
			e.StopPropagation()
		})
		el.On("custom-test", func(e dom.Event) {
			c.customCount++
		})
	}
}

func TestEvents(t *testing.T) {
	doc := SetupDOM(t)

	t.Run("Basic Event Handling", func(t *testing.T) {
		comp := &MockComponent{}
		comp.SetID("comp-basic-event")
		dom.Mount("root", comp)
		el, _ := dom.Get("comp-basic-event")

		clicked := false
		el.Click(func(e dom.Event) {
			clicked = true
		})

		rawEl := doc.Call("getElementById", "comp-basic-event")
		clickEvent := js.Global().Get("MouseEvent").New("click")
		rawEl.Call("dispatchEvent", clickEvent)

		if !clicked {
			t.Error("Click handler not called")
		}
	})

	t.Run("Complex Event Handling and Cleanup", func(t *testing.T) {
		comp := &EventComponent{}
		comp.SetID("comp-events")

		dom.Mount("root", comp)

		// Trigger events
		rawEl := doc.Call("getElementById", "comp-events")
		clickEvent := js.Global().Get("MouseEvent").New("click")
		rawEl.Call("dispatchEvent", clickEvent)

		customEvent := js.Global().Get("CustomEvent").New("custom-test")
		rawEl.Call("dispatchEvent", customEvent)

		if comp.clickCount != 1 || comp.customCount != 1 {
			t.Errorf("Events not triggered correctly: %d, %d", comp.clickCount, comp.customCount)
		}

		// Unmount and verify cleanup
		dom.Unmount(comp)

		// Trigger again
		// Note: Since element is removed from DOM, dispatching event on 'rawEl' (which is detached)
		// might still trigger listeners if they weren't removed.
		// However, standard DOM behavior says listeners on detached nodes still fire if event is dispatched to that node.
		// So we rely on 'Unmount' explicitly releasing the Go functions.
		// If released, invoking them should panic or error, OR if we are lucky, just not run.
		// But wait, js.Func.Release() makes the function invalid.
		// If the browser tries to call it, Go WASM runtime will likely print an error or panic.
		// We want to verify that our counts don't increase.

		rawEl.Call("dispatchEvent", clickEvent)
		rawEl.Call("dispatchEvent", customEvent)

		if comp.clickCount != 1 || comp.customCount != 1 {
			t.Errorf("Events triggered after unmount: %d, %d", comp.clickCount, comp.customCount)
		}
	})

	t.Run("Event Target Value", func(t *testing.T) {
		root, _ := dom.Get("root")
		root.AppendHTML(`<input id="test-input" value="initial">`)
		el, _ := dom.Get("test-input")

		var targetVal string
		el.On("input", func(e dom.Event) {
			targetVal = e.TargetValue()
		})

		rawEl := doc.Call("getElementById", "test-input")
		inputEvent := js.Global().Get("Event").New("input", map[string]interface{}{
			"bubbles": true,
		})
		rawEl.Call("dispatchEvent", inputEvent)

		if targetVal != "initial" {
			t.Errorf("Expected target value 'initial', got '%s'", targetVal)
		}
	})
}
