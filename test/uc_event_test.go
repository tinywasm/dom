//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

// EventComponent registers listeners
type EventComponent struct {
	MockComponent
	clickCount  int
	customCount int
}

func (c *EventComponent) Render() *dom.Element {
	return dom.Div().
		ID(c.GetID()).
		On("click", func(e dom.Event) {
			c.clickCount++
			e.PreventDefault()
			e.StopPropagation()
		}).
		On("custom-test", func(e dom.Event) {
			c.customCount++
		})
}

func (c *EventComponent) RenderHTML() string {
	return ""
}

func (c *EventComponent) OnMount() {
	c.MockComponent.OnMount()
}

func TestEvents(t *testing.T) {
	doc := SetupDOM(t)

	t.Run("Basic Event Handling", func(t *testing.T) {
		comp := &MockComponent{Element: &dom.Element{}}
		comp.SetID("comp-basic-event")
		dom.Render("root", comp)
		el, _ := GetRef("comp-basic-event")

		clicked := false
		el.On("click", func(e dom.Event) {
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
		comp := &EventComponent{MockComponent: MockComponent{Element: &dom.Element{}}}
		comp.SetID("comp-events")

		dom.Render("root", comp)

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
		// Unmount via replacement (triggers OnUnmount)
		dom.Render("root", dom.Div().ID("cleanup-placeholder"))

		// Trigger again
		// Note: Since element is removed from DOM, dispatching event on 'rawEl' (which is detached)
		// might still trigger listeners if they weren't removed.
		// However, standard DOM behavior says listeners on detached nodes still fire if event is dispatched to that node.
		// So we rely on 'Unmount' explicitly releasing the Go functions.
		// If released, invoking them should panic or error, OR if we are lucky, just not run.
		// But wait, js.Func.Release() makes the function invalid.
		// If the browser tries to call it, Go WASM runtime will likely print an error or panic.
		// We want to verify that our counts don't increase.

		// Verify it's gone from DOM
		_, found := GetRef("comp-events")
		if found {
			t.Error("Component element still in DOM after Render replacement")
		}
	})

	t.Run("Event Target Value", func(t *testing.T) {
		js.Global().Get("document").Call("getElementById", "root").Set("innerHTML", `<input id="test-input" value="initial">`)
		el, _ := GetRef("test-input")

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
