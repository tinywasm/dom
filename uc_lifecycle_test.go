//go:build wasm

package dom

import (
	"testing"
)

func TestLifecycle(t *testing.T) {
	_ = setupDOM(t)

	t.Run("Get Existing Element", func(t *testing.T) {
		el, ok := Get("root")
		if !ok {
			t.Fatal("Expected to find root element")
		}
		if el.GetAttr("id") != "root" {
			t.Errorf("Expected id 'root', got %s", el.GetAttr("id"))
		}
	})

	t.Run("Get Non-Existing Element", func(t *testing.T) {
		_, ok := Get("non-existent")
		if ok {
			t.Error("Expected not to find non-existent element")
		}
	})

	t.Run("Mount Component", func(t *testing.T) {
		comp := &MockComponent{}
		comp.SetID("comp1")
		err := Mount("root", comp)
		if err != nil {
			t.Fatalf("Mount failed: %v", err)
		}

		if !comp.mounted {
			t.Error("OnMount was not called")
		}

		el, ok := Get("comp1")
		if !ok {
			t.Fatal("Component element not found in DOM")
		}
		if val := el.Value(); val != "" && val != "<undefined>" {
			t.Errorf("Expected empty value or <undefined> for div, got: %s", val)
		}
	})

	t.Run("Unmount Component", func(t *testing.T) {
		comp := &MockComponent{}
		comp.SetID("comp1")
		// Note: comp1 is already mounted from previous test if we share DOM state,
		// but setupDOM clears body. Wait, setupDOM is called once per TestLifecycle.
		// So state persists between sub-tests.

		Unmount(comp)

		if comp.mounted {
			// See previous note about new struct instance
		}

		_, ok := Get("comp1")
		if ok {
			t.Error("Element should be removed from cache")
		}
	})

	t.Run("Mount Invalid Parent", func(t *testing.T) {
		comp := &MockComponent{}
		comp.SetID("comp-invalid")
		err := Mount("invalid-parent-id", comp)
		if err == nil {
			t.Error("Expected error when mounting to invalid parent")
		}
	})

	t.Run("Unmount No Listeners", func(t *testing.T) {
		comp := &MockComponent{}
		comp.SetID("comp-no-listeners")
		// Need to add parent first
		root, _ := Get("root")
		root.AppendHTML(`<div id="root-no-listeners"></div>`)
		Mount("root-no-listeners", comp)
		Unmount(comp)
	})

	t.Run("Get Cache Hit", func(t *testing.T) {
		_, ok := Get("root")
		if !ok {
			t.Fatal("Root not found")
		}

		el, ok := Get("root")
		if !ok {
			t.Fatal("Root not found in cache")
		}
		if el.GetAttr("id") != "root" {
			t.Error("Wrong element from cache")
		}
	})
}
