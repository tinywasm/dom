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

// sharedChild is the shape that produced the bug in the wild: a shell held ONE
// *ThemeToggle in a field and rendered it into both its header menu and its
// drawer menu. Both copies appeared, both looked right, and the second did
// nothing when clicked — its handler had been wired to the first, because the
// runtime resolves handlers by id and both nodes carried the same one.
type sharedChild struct{ Element }

func newSharedChild(label string) *sharedChild {
	s := &sharedChild{Element: *NewElement("button")}
	s.Text(label)
	return s
}

func TestOneComponentInstanceRenderedInTwoPlacesIsCaught(t *testing.T) {
	toggle := newSharedChild("theme")
	toggle.SetID("tg") // the id it was minted on its first render

	shell := NewElement("div").
		Child(NewElement("header").Child(toggle)).
		Child(NewElement("nav").Child(toggle))

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("rendering one component instance in two places must not pass silently")
		}
		msg := r.(error).Error()
		for _, want := range []string{"written twice", "tg", "factory"} {
			if !strings.Contains(msg, want) {
				t.Errorf("panic should mention %q, got: %s", want, msg)
			}
		}
	}()

	_ = shell.String()
}

// The fix a consumer is expected to make: a factory instead of a shared slot.
func TestAFactoryPerPlaceRendersBothCopies(t *testing.T) {
	build := func() Component { return newSharedChild("theme") }

	shell := NewElement("div").
		Child(NewElement("header").Child(build())).
		Child(NewElement("nav").Child(build()))

	if got := strings.Count(shell.String(), "theme"); got != 2 {
		t.Fatalf("expected both copies rendered, got %d", got)
	}
}

// Two DIFFERENT elements given the same explicit id is the other way in — a
// layout that stamps a module id onto both a section and the component mounted
// inside it. Same illegal state, same consequence.
func TestTwoElementsWithTheSameExplicitIdAreCaught(t *testing.T) {
	shell := NewElement("main").
		Child(NewElement("section").ID("mod1")).
		Child(NewElement("div").ID("mod1"))

	defer func() {
		if recover() == nil {
			t.Fatal("two nodes sharing an explicit id must not pass silently")
		}
	}()

	_ = shell.String()
}

// Re-rendering the SAME tree re-emits the same ids and must stay legal — the
// check is scoped to one pass, not to the lifetime of the elements.
func TestRepeatedRendersOfTheSameTreeAreLegal(t *testing.T) {
	tree := NewElement("div").ID("root").Child(NewElement("span").ID("leaf"))

	first := tree.String()
	second := tree.String()

	if first != second {
		t.Fatalf("render not idempotent:\n%s\n%s", first, second)
	}
}
