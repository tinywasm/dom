//go:build wasm

package dom

import (
	"syscall/js"
	"testing"
)

func TestInternalWasm(t *testing.T) {
	td := &tinyDOM{}
	d := newDom(td).(*domWasm)

	t.Run("splitEventKey", func(t *testing.T) {
		parts := d.splitEventKey("id::type")
		if len(parts) != 2 || parts[0] != "id" || parts[1] != "type" {
			t.Errorf("splitEventKey failed: %v", parts)
		}
	})

	t.Run("Element Cache", func(t *testing.T) {
		d.removeFromElementCache("none")
		d.elementCache = append(d.elementCache,
			struct {
				id  string
				val js.Value
			}{"id1", js.Null()},
			struct {
				id  string
				val js.Value
			}{"id2", js.Null()},
		)
		d.removeFromElementCache("id1")
		d.removeFromElementCache("id2")
	})

	t.Run("Component Tracking", func(t *testing.T) {
		d.untrackComponent("none")
		cl1 := &comp{id: "id1"}
		cl2 := &comp{id: "id2"}
		d.trackComponent(cl1)
		d.trackComponent(cl1)

		d.mountedComponents = append(d.mountedComponents, struct {
			id   string
			comp Component
		}{"id2", cl2})
		d.untrackComponent("id1")
		d.untrackComponent("id2")
	})

	t.Run("Children Map", func(t *testing.T) {
		d.trackChildren("p1", []Component{&comp{id: "n1"}})

		d.cleanupChildren("none")
		d.childrenMap = append(d.childrenMap,
			struct {
				parentID string
				childIDs []string
			}{"p-cleanup", []string{"c1"}},
			struct {
				parentID string
				childIDs []string
			}{"p-other", []string{"o1"}},
		)
		d.cleanupChildren("p-cleanup")
		d.cleanupChildren("p-other")
	})

	t.Run("Listeners", func(t *testing.T) {
		d.cleanupListeners("none")
		d.componentListeners = append(d.componentListeners, struct {
			id   string
			keys []string
		}{"id1", []string{"id1::click"}})

		d.eventFuncs = append(d.eventFuncs,
			struct {
				key string
				val js.Value
				fn  js.Func
			}{"id1::click", js.Null(), js.Func{}},
		)

		d.cleanupListeners("id1")
	})

	t.Run("Lifecycle", func(t *testing.T) {
		m := &mountableComp{}
		m.id = "m1"
		d.mountRecursive(m)

		um := &unmountableComp{}
		um.id = "um1"
		d.unmountRecursive(um)
	})

	t.Run("unmountRecursive Complex", func(t *testing.T) {
		child1 := &comp{id: "child1"}
		parent := &comp{id: "parent", kids: []Component{child1}}

		d.mountedComponents = append(d.mountedComponents,
			struct {
				id   string
				comp Component
			}{"child1", child1},
		)
		d.childrenMap = append(d.childrenMap, struct {
			parentID string
			childIDs []string
		}{
			"parent", []string{"child1", "missing"},
		})

		d.unmountRecursive(parent)
	})

	t.Run("renderToHTML", func(t *testing.T) {
		childComp := &comp{id: "child-comp"}
		parent := Div(childComp, "text")
		var comps []Component
		_ = d.renderToHTML(parent, &comps, "parent-id")

		// Verify child component root has ID injected
		// The comp.RenderHTML returns "<div></div>", but we are using factory Div which returns *Element
		// Wait, comp doesn't implement ViewRenderer or elementNode, it just has RenderHTML.
		// Let's use a better mock.
		if len(comps) != 1 {
			t.Errorf("expected 1 child component, got %d", len(comps))
		}

		// Test with ViewRenderer
		vr := &viewRendererComp{id: "vr-1"}
		parent2 := Div(vr)
		var comps2 []Component
		html2 := d.renderToHTML(parent2, &comps2, "parent-id")

		expected := "<div><div id='vr-1'></div></div>"
		if html2 != expected {
			t.Errorf("expected %q, got %q", expected, html2)
		}
	})

	t.Run("Factories", func(t *testing.T) {
		_ = Div("e1")
		_ = Span("e2", "p1")
		_ = Button("t1")
		_ = P("t2", "p1")
	})

	t.Run("For Method", func(t *testing.T) {
		input := Input("text")
		label := Label().For(input)

		id := input.GetID()
		if id == "" {
			t.Fatal("input ID should not be empty")
		}

		forAttr := ""
		for _, attr := range label.attrs {
			if attr.Key == "for" {
				forAttr = attr.Value
				break
			}
		}

		if forAttr != id {
			t.Errorf("expected for=%q, got %q", id, forAttr)
		}
	})
}

type comp struct {
	id   string
	kids []Component
}

func (c *comp) GetID() string         { return c.id }
func (c *comp) SetID(id string)       { c.id = id }
func (c *comp) RenderHTML() string    { return "<div></div>" }
func (c *comp) Children() []Component { return c.kids }

type mountableComp struct {
	comp
	mounted bool
}

func (c *mountableComp) OnMount() { c.mounted = true }

type unmountableComp struct {
	comp
	unmounted bool
}

func (c *unmountableComp) OnUnmount() { c.unmounted = true }

type viewRendererComp struct {
	id string
}

func (c *viewRendererComp) GetID() string         { return c.id }
func (c *viewRendererComp) SetID(id string)       { c.id = id }
func (c *viewRendererComp) RenderHTML() string    { return "<div></div>" }
func (c *viewRendererComp) Children() []Component { return nil }
func (c *viewRendererComp) Render() *Element      { return Div() }
