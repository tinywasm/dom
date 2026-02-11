//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

// RenderableComp implements ViewRenderer for testing the full rendering pipeline.
type RenderableComp struct {
	*dom.Element
	label string
}

func (c *RenderableComp) Render() *dom.Element {
	return dom.Div().
		Class("test-comp").
		Add(
			dom.Span().
				ID(c.GetID() + "-label").
				Text(c.label),
		)
}

func (c *RenderableComp) RenderHTML() string { return "" }

// StaticTestComp implements only RenderHTML for testing the fallback path.
type StaticTestComp struct {
	*dom.Element
}

func (c *StaticTestComp) RenderHTML() string {
	return `<div id="` + c.GetID() + `"><span>static</span></div>`
}

func TestBodyHeadResolution(t *testing.T) {
	_ = SetupDOM(t)

	// Note: We avoid testing dom.Render("body", ...) because it sets innerHTML,
	// potentially wiping out the test runner's scripts/UI and casing a timeout.
	// Testing dom.Append("body", ...) is sufficient to verify that getElement("body") works.

	t.Run("Append ViewRenderer to body", func(t *testing.T) {
		comp := &RenderableComp{Element: &dom.Element{}, label: "body-append"}
		comp.SetID("body-append-vr")
		err := dom.Append("body", comp)
		if err != nil {
			t.Fatalf("Append to body failed: %v", err)
		}

		el, ok := GetRef("body-append-vr-label")
		if !ok {
			t.Fatal("Appended component not found in body")
		}
		if val := el.GetAttr("id"); val != "body-append-vr-label" {
			t.Errorf("Expected id 'body-append-vr-label', got '%s'", val)
		}
	})

	t.Run("Append RenderHTML to body", func(t *testing.T) {
		comp := &StaticTestComp{Element: &dom.Element{}}
		comp.SetID("body-static")
		err := dom.Append("body", comp)
		if err != nil {
			t.Fatalf("Append static to body failed: %v", err)
		}

		_, ok := GetRef("body-static")
		if !ok {
			t.Fatal("Appended static component not found in body")
		}
	})

	t.Run("Append to head", func(t *testing.T) {
		// Use a meta tag wrapper or just a hidden div to test head injection
		comp := &StaticTestComp{Element: &dom.Element{}}
		comp.SetID("head-item")
		// Override RenderHTML to be valid head content
		// (Though browsers are lenient, keeping it simple)
		err := dom.Append("head", comp)
		if err != nil {
			t.Fatalf("Append to head failed: %v", err)
		}

		_, ok := GetRef("head-item")
		if !ok {
			t.Fatal("Component not found in head")
		}
	})
}

// TestAutoID verifies that components get auto-generated IDs when none is set.
func TestAutoID(t *testing.T) {
	_ = SetupDOM(t)

	comp := &RenderableComp{Element: &dom.Element{}, label: "auto"}
	// Intentionally do NOT set an ID
	err := dom.Render("root", comp)
	if err != nil {
		t.Fatalf("Render with auto-ID failed: %v", err)
	}

	id := comp.GetID()
	if id == "" {
		t.Fatal("Expected auto-generated ID, got empty string")
	}
	_ = fmt.Sprint(id) // use fmt to avoid import error
}
