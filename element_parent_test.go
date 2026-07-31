package dom

import (
	"strings"
	"testing"
)

// An element has exactly one parent. Ids are minted per element, so the same
// pointer under two parents renders twice with ONE id, and every handler and
// binding wires to whichever copy the runtime resolved — the other is inert.
//
// This was found the expensive way: a shell passed one theme-toggle component
// into two menus, it appeared in both, and the second did nothing when clicked.
func TestChildRejectsAnElementThatAlreadyHasAParent(t *testing.T) {
	shared := NewElement("button").Text("toggle")

	NewElement("div").Child(shared)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected sharing one element between two parents to panic")
		}
		msg := r.(error).Error()
		for _, want := range []string{"one element has one parent", "button"} {
			if !strings.Contains(msg, want) {
				t.Errorf("panic should say %q, got: %s", want, msg)
			}
		}
	}()

	NewElement("div").Child(shared)
}

func TestChildAcceptsSeparateInstances(t *testing.T) {
	// The fix a consumer is expected to make: build one per parent.
	build := func() *Element { return NewElement("button").Text("toggle") }

	a := NewElement("div").Child(build())
	b := NewElement("div").Child(build())

	if len(a.children) != 1 || len(b.children) != 1 {
		t.Fatalf("expected one child each, got %d and %d", len(a.children), len(b.children))
	}
}
