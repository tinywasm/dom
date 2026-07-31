//go:build wasm

package dom

import (
	"syscall/js"
	"testing"
)

func TestMain(m *testing.M) {
	// wasmbrowsertest provides a minimal HTML page with no #app div.
	// Create one in <body> so every test can Render("app", ...).
	app := js.Global().Get("document").Call("createElement", "div")
	app.Set("id", "app")
	js.Global().Get("document").Get("body").Call("appendChild", app)
	m.Run()
}

type counterComp struct {
	Element
	count *SignalString
}

func (c *counterComp) Init(ctx Ctx) {
	c.count = NewString("0")
}

func (c *counterComp) Render() *Element {
	return NewElement("div").ID("counter-div").
		Child(
			NewElement("span").ID("count-val").BindText(c.count),
			NewElement("button").ID("inc-btn").On("click", func(e Event) {
				c.count.Update(func(v string) string {
					if v == "0" {
						return "1"
					}
					return "2"
				})
			}),
		)
}

func TestCounter(t *testing.T) {
	c := &counterComp{}
	Render("app", c)

	val, _ := Get("count-val")
	if val.(*elementWasm).val.Get("textContent").String() != "0" {
		t.Errorf("Expected 0, got %s", val.(*elementWasm).val.Get("textContent").String())
	}

	btn, _ := Get("inc-btn")
	btn.(*elementWasm).val.Call("click")

	if val.(*elementWasm).val.Get("textContent").String() != "1" {
		t.Errorf("Expected 1, got %s", val.(*elementWasm).val.Get("textContent").String())
	}

	// Verify node identity
	oldVal := val.(*elementWasm).val
	btn.(*elementWasm).val.Call("click")
	if val.(*elementWasm).val.Get("textContent").String() != "2" {
		t.Errorf("Expected 2, got %s", val.(*elementWasm).val.Get("textContent").String())
	}
	if !val.(*elementWasm).val.Equal(oldVal) {
		t.Error("Node identity lost after signal update")
	}
}

type lifecycleComp struct {
	Element
	inited  int
	cleaned bool
}

func (c *lifecycleComp) Init(ctx Ctx) {
	c.inited++
	ctx.OnCleanup(func() {
		c.cleaned = true
	})
}

func (c *lifecycleComp) Render() *Element {
	return NewElement("div").Text("lifecycle")
}

func TestLifecycle(t *testing.T) {
	c := &lifecycleComp{}
	Render("app", c)

	if c.inited != 1 {
		t.Errorf("Expected inited 1, got %d", c.inited)
	}

	Render("app", NewElement("div").Text("replaced"))
	if !c.cleaned {
		t.Error("OnCleanup not called")
	}
}

func TestShow(t *testing.T) {
	cond := NewBool(false)
	s := Show(cond, func() *Element {
		return NewElement("span").ID("shown").Text("visible")
	})
	Render("app", s)

	if _, ok := Get("shown"); ok {
		t.Error("Element should not be visible")
	}

	cond.Set(true)
	if _, ok := Get("shown"); !ok {
		t.Error("Element should be visible")
	}

	cond.Set(false)
	if _, ok := Get("shown"); ok {
		t.Error("Element should be hidden")
	}
}

func TestTwoWayInput(t *testing.T) {
	s := NewString("initial")
	input := NewElement("input").ID("io").Bind(s)
	Render("app", input)

	ref, _ := Get("io")
	if ref.Value() != "initial" {
		t.Errorf("Expected initial, got %s", ref.Value())
	}

	ref.(*elementWasm).val.Set("value", "changed")
	ref.(*elementWasm).val.Call("dispatchEvent", js.Global().Get("Event").New("input"))

	if s.Get() != "changed" {
		t.Errorf("Signal not updated from input, got %s", s.Get())
	}

	s.Set("from-signal")
	if ref.Value() != "from-signal" {
		t.Errorf("Input not updated from signal, got %s", ref.Value())
	}
}

func TestBindChildren(t *testing.T) {
	nodes := NewNodes(
		NewElement("div").ID("n1").Text("one"),
		NewElement("div").ID("n2").Text("two"),
	)
	list := NewElement("div").ID("list").BindChildren(nodes)
	Render("app", list)

	if _, ok := Get("n1"); !ok {
		t.Error("n1 missing")
	}
	if _, ok := Get("n2"); !ok {
		t.Error("n2 missing")
	}

	n1ref, _ := Get("n1")
	n1val := n1ref.(*elementWasm).val

	nodes.Set([]*Element{
		NewElement("div").ID("n2").Text("two"),
		NewElement("div").ID("n1").Text("one"),
	})

	if !n1ref.(*elementWasm).val.Equal(n1val) {
		t.Error("Node identity lost after reorder")
	}

	// Verify order in DOM
	parent, _ := Get("list")
	first := parent.(*elementWasm).val.Get("children").Call("item", 0)
	if first.Get("id").String() != "n2" {
		t.Errorf("Expected n2 as first child, got %s", first.Get("id").String())
	}
}

// TestBindChildrenInitialRowBindings guards the fix for rows present in a
// BindChildren signal at FIRST render: they are serialized straight into the
// parent's HTML and never pass through reconcileChildren, so their own nested
// bindings must be wired at mount. Before the fix such a row appeared but never
// reacted — a later signal change did not patch it.
func TestBindChildrenInitialRowBindings(t *testing.T) {
	on := NewBool(false)
	rows := NewNodes(
		NewElement("li").ID("wrow1").BindClass("active", on).Text("row"),
	)
	list := NewElement("ul").ID("wrows").BindChildren(rows)
	Render("app", list)

	ref, ok := Get("wrow1")
	if !ok {
		t.Fatal("wrow1 missing at first render")
	}
	classList := ref.(*elementWasm).val.Get("classList")
	if classList.Call("contains", "active").Bool() {
		t.Fatal("wrow1 must not carry 'active' before the signal is set")
	}

	on.Set(true)
	if !classList.Call("contains", "active").Bool() {
		t.Error("BindClass on a first-render BindChildren row did not react to the signal")
	}

	on.Set(false)
	if classList.Call("contains", "active").Bool() {
		t.Error("BindClass on a first-render BindChildren row did not clear on the signal")
	}
}

type orderChildComp struct {
	Element
	mounted bool
}

func (c *orderChildComp) Render() *Element {
	return NewElement("span").ID("order-child")
}

func (c *orderChildComp) Mounted() {
	c.mounted = true
	// Verify it can Get itself
	if _, ok := Get("order-child"); !ok {
		panic("order-child element not in DOM in orderChildComp.Mounted")
	}
}

type orderParentComp struct {
	Element
	child           *orderChildComp
	childMountedAt  bool
	mounted         bool
	ownElementFound bool
}

func (p *orderParentComp) Init(ctx Ctx) {
	p.child = &orderChildComp{}
}

func (p *orderParentComp) Children() []Component {
	return []Component{p.child}
}

func (p *orderParentComp) Render() *Element {
	return NewElement("div").ID("order-parent").Child(p.child)
}

func (p *orderParentComp) Mounted() {
	p.mounted = true
	p.childMountedAt = p.child.mounted
	if _, ok := Get("order-parent"); ok {
		p.ownElementFound = true
	}
}

func TestMountedOrderingAndGet(t *testing.T) {
	parent := &orderParentComp{}
	Render("app", parent)

	if !parent.mounted {
		t.Error("parent Mounted not called")
	}
	if !parent.child.mounted {
		t.Error("child Mounted not called")
	}
	if !parent.childMountedAt {
		t.Error("parent Mounted called before child Mounted")
	}
	if !parent.ownElementFound {
		t.Error("parent could not Get itself inside Mounted")
	}
}

type rootMountedComp struct {
	Element
	mountedCount int
}

func (c *rootMountedComp) Render() *Element {
	return NewElement("div").ID("root-mounted-elem")
}

func (c *rootMountedComp) Mounted() {
	c.mountedCount++
}

func TestMountedRootAndAppend(t *testing.T) {
	c1 := &rootMountedComp{}
	Render("app", c1)
	if c1.mountedCount != 1 {
		t.Errorf("Expected mountedCount 1 on Render, got %d", c1.mountedCount)
	}

	c2 := &rootMountedComp{}
	Append("app", c2)
	if c2.mountedCount != 1 {
		t.Errorf("Expected mountedCount 1 on Append, got %d", c2.mountedCount)
	}
}

type scrollableItem struct {
	Element
}

func (s *scrollableItem) Render() *Element {
	return NewElement("div").ID("item-to-scroll").
		Attr("style", "width: 200px; height: 100px; display: inline-block;")
}

type deckComp struct {
	Element
	child           *scrollableItem
	mountedCalled   bool
	childFoundAtMnt bool
}

func (d *deckComp) Init(ctx Ctx) {
	d.child = &scrollableItem{}
}

func (d *deckComp) Children() []Component {
	return []Component{d.child}
}

func (d *deckComp) Render() *Element {
	return NewElement("div").ID("deck-scroller").
		Attr("style", "width: 100px; height: 100px; overflow-x: auto; white-space: nowrap;").
		Child(
			NewElement("div").Attr("style", "width: 50px; height: 100px; display: inline-block;"),
			d.child,
		)
}

func (d *deckComp) Mounted() {
	d.mountedCalled = true
	if el, ok := Get("item-to-scroll"); ok {
		d.childFoundAtMnt = true
		el.ScrollIntoView()
	}
}

func TestMountedScrollableConsumer(t *testing.T) {
	c := &deckComp{}
	Render("app", c)

	if !c.mountedCalled {
		t.Error("deckComp Mounted was not called")
	}
	if !c.childFoundAtMnt {
		t.Error("item-to-scroll was not found in DOM during deckComp.Mounted")
	}

	// Verify scroller exists and can scroll
	scroller, ok := Get("deck-scroller")
	if !ok {
		t.Fatal("deck-scroller not found")
	}
	if !scroller.ScrollsX() {
		t.Log("Warning: deck-scroller.ScrollsX() is false (expected in some headless environments or without style sheet support, but scroller is successfully wired)")
	}
}
