//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

// mountCountComp counts how many times OnMount is called — used to detect double-mount.
type mountCountComp struct {
	*dom.Element
	MountCount int
}

func (c *mountCountComp) Render() *dom.Element { return c.Element }
func (c *mountCountComp) AsElement() *dom.Element { return c.Element }
func (c *mountCountComp) RenderHTML() string { return c.Element.RenderHTML() }
func (c *mountCountComp) OnMount() { c.MountCount++ }

// TestRender_OnMount_CalledOnce verifies that a Mountable component nested inside a
// container rendered via dom.Render has OnMount called exactly once.
//
// Regression: dom.Render called both mountRecursive(component) (which recurses through
// Children()) AND a separate loop over children collected by renderToHTML. Components
// appearing in both paths had OnMount called twice, causing double event registration
// (e.g. two submit listeners → two POST requests per form submission).
func TestRender_OnMount_CalledOnce(t *testing.T) {
	SetupDOM(t)

	child := &mountCountComp{Element: dom.Div().ID("mount-child")}

	container := dom.Div(child).ID("mount-container")

	if err := dom.Render("root", container); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if child.MountCount != 1 {
		t.Errorf("OnMount called %d times, want 1 — double-mount bug", child.MountCount)
	}
}

// TestRender_OnMount_MultipleChildren verifies that each nested Mountable child
// has OnMount called exactly once, not once per nesting path.
func TestRender_OnMount_MultipleChildren(t *testing.T) {
	SetupDOM(t)

	a := &mountCountComp{Element: dom.Div().ID("mc-a")}
	b := &mountCountComp{Element: dom.Div().ID("mc-b")}
	c := &mountCountComp{Element: dom.Div().ID("mc-c")}

	container := dom.Div(a, b, c).ID("mc-container")

	if err := dom.Render("root", container); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	for name, comp := range map[string]*mountCountComp{"a": a, "b": b, "c": c} {
		if comp.MountCount != 1 {
			t.Errorf("child %s: OnMount called %d times, want 1", name, comp.MountCount)
		}
	}
}

// TestAppend_OnMount_CalledOnce verifies that a Mountable component nested inside a
// container appended via dom.Append has OnMount called exactly once.
func TestAppend_OnMount_CalledOnce(t *testing.T) {
	SetupDOM(t)

	child := &mountCountComp{Element: dom.Div().ID("append-child")}
	container := dom.Div(child).ID("append-container")

	// Ensure target exists in SetupDOM environment (typically "root" is available)
	if err := dom.Append("root", container); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if child.MountCount != 1 {
		t.Errorf("Append OnMount called %d times, want 1", child.MountCount)
	}
}

