//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

type DynamicComp struct {
	dom.BaseComponent
}

func (c *DynamicComp) Render() dom.Node {
	return dom.Div().ID(c.GetID()).ToNode()
}

type StaticComp struct {
	dom.BaseComponent
}

func (c *StaticComp) RenderHTML() string {
	return `<div id="` + c.GetID() + `">Static Content</div>`
}

func TestHybridRendering(t *testing.T) {
	_ = dom.SetupDOM(t)

	// Test ViewRenderer (DSL)
	dc := &DynamicComp{}
	dc.SetID("dynamic")
	err := dom.Render("root", dc)
	if err != nil {
		t.Fatalf("Render failed for ViewRenderer: %v", err)
	}
	if _, ok := dom.Get("dynamic"); !ok {
		t.Error("Dynamic component failed to render")
	}

	// Test HTMLRenderer (string)
	sc := &StaticComp{}
	sc.SetID("static")
	// Use Append to keep dynamic component
	err = dom.Append("root", sc)
	if err != nil {
		t.Fatalf("Append failed for HTMLRenderer: %v", err)
	}
	if _, ok := dom.Get("static"); !ok {
		t.Error("Static component failed to render")
	}
}
