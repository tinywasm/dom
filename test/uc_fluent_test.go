//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

func TestFluentBuilder(t *testing.T) {
	_ = SetupDOM(t)

	t.Run("Chainable Methods", func(t *testing.T) {
		el := dom.Div().
			ID("test-id").
			Class("cls1").
			Class("cls2").
			Attr("data-foo", "bar").
			Text("Hello")

		// Verify builder state by converting to Node
		node := el.ToNode()

		if node.Tag != "div" {
			t.Errorf("Expected tag div, got %s", node.Tag)
		}

		// Check attributes
		idFound := false
		classFound := false
		attrFound := false

		for _, a := range node.Attrs {
			if a.Key == "id" && a.Value == "test-id" {
				idFound = true
			}
			if a.Key == "class" && a.Value == "cls1 cls2" {
				classFound = true
			}
			if a.Key == "data-foo" && a.Value == "bar" {
				attrFound = true
			}
		}

		if !idFound {
			t.Error("ID attribute not found or incorrect")
		}
		if !classFound {
			t.Error("Class attribute not found or incorrect")
		}
		if !attrFound {
			t.Error("Custom attribute not found or incorrect")
		}

		// Check children
		if len(node.Children) != 1 {
			t.Errorf("Expected 1 child, got %d", len(node.Children))
		}
		if txt, ok := node.Children[0].(string); !ok || txt != "Hello" {
			t.Error("Child text incorrect")
		}
	})

	t.Run("Nested Builders", func(t *testing.T) {
		parent := dom.Div().
			ID("parent").
			Append(
				dom.Span().ID("child").Text("Child"),
			)

		node := parent.ToNode()
		if len(node.Children) != 1 {
			t.Fatal("Expected 1 child")
		}

		// Children are Nodes now (converted by ToNode)
		childNode, ok := node.Children[0].(dom.Node)
		if !ok {
			t.Fatal("Child is not a Node")
		}

		if childNode.Tag != "span" {
			t.Errorf("Expected child tag span, got %s", childNode.Tag)
		}
	})
	t.Run("Variadic Add", func(t *testing.T) {
		el := dom.Div().Add(
			dom.Span().Text("One"),
			dom.Span().Text("Two"),
			dom.Span().Text("Three"),
		)

		node := el.ToNode()
		if len(node.Children) != 3 {
			t.Errorf("Expected 3 children, got %d", len(node.Children))
		}
	})
}
