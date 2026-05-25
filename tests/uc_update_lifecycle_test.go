//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

type trackableComp struct {
	*dom.Element
	mountCount   int
	unmountCount int
}

func (c *trackableComp) RenderHTML() string  { return c.Element.RenderHTML() }
func (c *trackableComp) AsElement() *dom.Element { return c.Element }
func (c *trackableComp) Render() *dom.Element    { return c.Element }
func (c *trackableComp) OnMount()                { c.mountCount++ }
func (c *trackableComp) OnUnmount()              { c.unmountCount++ }

// TestUpdateLifecycle_UnmountBeforeRemount verifies that dom calls OnUnmount
// before OnMount when a component is updated via dom.Render.
// Without this, components that register DOM listeners in OnMount accumulate
// duplicate listeners on each update, causing handlers to fire multiple times
// (e.g. a form submitting multiple POST requests on a single user action).
func TestUpdateLifecycle_UnmountBeforeRemount(t *testing.T) {
	SetupDOM(t)

	comp := &trackableComp{Element: dom.Div()}
	comp.SetID("trackable")

	// First render — mounts the component
	if err := dom.Render("root", comp); err != nil {
		t.Fatalf("first Render failed: %v", err)
	}
	if comp.mountCount != 1 {
		t.Errorf("expected mountCount=1 after first render, got %d", comp.mountCount)
	}
	if comp.unmountCount != 0 {
		t.Errorf("expected unmountCount=0 after first render, got %d", comp.unmountCount)
	}

	// Second render (update) — must call OnUnmount then OnMount
	if err := dom.Render("root", comp); err != nil {
		t.Fatalf("second Render failed: %v", err)
	}
	if comp.unmountCount != 1 {
		t.Errorf("expected unmountCount=1 before re-mount on update, got %d (bug: OnUnmount not called)", comp.unmountCount)
	}
	if comp.mountCount != 2 {
		t.Errorf("expected mountCount=2 after update, got %d", comp.mountCount)
	}
}
