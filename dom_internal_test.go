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
		_ = d.renderToHTML(parent, &comps)
	})

	t.Run("Factories", func(t *testing.T) {
		_ = Email("e1")
		_ = Email("e2", "p1")
		_ = Textarea("t1")
		_ = Textarea("t2", "p1")
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
