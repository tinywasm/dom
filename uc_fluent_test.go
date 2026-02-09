//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

func TestFluentBuilder(t *testing.T) {
	// Test chainable API
	el := dom.Div().
		ID("test").
		Class("container").
		Append(dom.Button().Text("Click"))

	if el.GetID() != "test" {
		t.Error("ID not set")
	}

	node := el.ToNode()
	if node.Tag != "div" {
		t.Errorf("Expected tag div, got %s", node.Tag)
	}

	// Check attrs
	hasID := false
	hasClass := false
	for _, attr := range node.Attrs {
		if attr.Key == "id" && attr.Value == "test" {
			hasID = true
		}
		if attr.Key == "class" && attr.Value == "container" {
			hasClass = true
		}
	}

	if !hasID {
		t.Error("ID attribute missing in Node")
	}
	if !hasClass {
		t.Error("Class attribute missing in Node")
	}

	// Check children
	if len(node.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(node.Children))
	}
}
