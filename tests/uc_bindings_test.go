//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	. "github.com/tinywasm/dom"
)

// setupBindRoot prepares a clean #bind-root div in the page body.
func setupBindRoot() {
	doc := js.Global().Get("document")
	existing := doc.Call("getElementById", "bind-root")
	if !existing.IsNull() {
		existing.Set("innerHTML", "")
		return
	}
	root := doc.Call("createElement", "div")
	root.Set("id", "bind-root")
	doc.Get("body").Call("appendChild", root)
}

func queryText(selector string) string {
	el := js.Global().Get("document").Call("querySelector", selector)
	if el.IsNull() || el.IsUndefined() {
		return "<not found>"
	}
	return el.Get("textContent").String()
}

// BindTextComp — regression: wireBindings called Render() a second time,
// producing new auto-IDs that don't match the DOM, so BindText subscriptions
// silently targeted phantom elements and the DOM never updated on signal.Set().
type BindTextComp struct {
	Element
	label *SignalString
}

func (c *BindTextComp) Init(_ Ctx) { c.label = NewString("initial") }
func (c *BindTextComp) Render() *Element {
	return NewElement("div").ID(c.GetID()).
		Child(NewElement("span").ID("btc-span").BindText(c.label))
}

func TestBindText_UpdatesDOM(t *testing.T) {
	setupBindRoot()
	comp := &BindTextComp{}
	comp.SetID("btc-root")
	if err := Render("bind-root", comp); err != nil {
		t.Fatalf("Render: %v", err)
	}

	if got := queryText("#btc-span"); got != "initial" {
		t.Fatalf("before Set: want 'initial', got %q", got)
	}

	comp.label.Set("updated")

	if got := queryText("#btc-span"); got != "updated" {
		t.Errorf("after Set: want 'updated', got %q — BindText subscription targeting wrong ID (double-Render bug)", got)
	}
}

// ChildBindComp / ParentWithChild — regression: child components embedded
// inside a parent's Render() never had wireBindings called for them, so their
// BindText/BindAttr bindings were never wired and signals had no DOM effect.
type ChildBindComp struct {
	Element
	value *SignalString
}

func (c *ChildBindComp) Init(_ Ctx) { c.value = NewString("child-initial") }
func (c *ChildBindComp) Render() *Element {
	return NewElement("p").ID(c.GetID()).
		Child(NewElement("span").ID("cbc-span").BindText(c.value))
}

type ParentWithChild struct {
	Element
	child ChildBindComp
}

func (p *ParentWithChild) Render() *Element {
	p.child.SetID("cbc-p")
	return NewElement("div").ID(p.GetID()).Child(&p.child)
}

func TestBindText_ChildComponent_UpdatesDOM(t *testing.T) {
	setupBindRoot()
	parent := &ParentWithChild{}
	parent.SetID("pwc-root")
	if err := Render("bind-root", parent); err != nil {
		t.Fatalf("Render: %v", err)
	}

	if got := queryText("#cbc-span"); got != "child-initial" {
		t.Fatalf("before Set: want 'child-initial', got %q", got)
	}

	parent.child.value.Set("child-updated")

	if got := queryText("#cbc-span"); got != "child-updated" {
		t.Errorf("after Set: want 'child-updated', got %q — child component bindings not wired (mountRecursive missing wireBindings)", got)
	}
}
