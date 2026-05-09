//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
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

		// Verify builder state by checking rendered HTML
		html := el.RenderHTML()

		if html == "" {
			t.Error("RenderHTML returned empty string")
		}

		// Look for expected substrings since it's easier than full parsing here
		expected := []string{
			"<div", "id='test-id'", "class='cls1 cls2'", "data-foo='bar'", ">Hello</div>",
		}
		for _, exp := range expected {
			if !fmt.Contains(html, exp) {
				t.Errorf("Expected HTML to contain %q, but got %q", exp, html)
			}
		}
	})

	t.Run("Nested Elements", func(t *testing.T) {
		parent := dom.Div().
			ID("parent").
			Add(
				dom.Span().ID("child").Text("Child"),
			)

		html := parent.RenderHTML()
		expected := []string{
			"id='parent'", "<span id='child'>Child</span>",
		}
		for _, exp := range expected {
			if !fmt.Contains(html, exp) {
				t.Errorf("Expected HTML to contain %q, but got %q", exp, html)
			}
		}
	})
	t.Run("Variadic Add", func(t *testing.T) {
		el := dom.Div().Add(
			dom.Span().Text("One"),
			dom.Span().Text("Two"),
			dom.Span().Text("Three"),
		)

		html := el.RenderHTML()
		expected := []string{"One", "Two", "Three"}
		for _, exp := range expected {
			if !fmt.Contains(html, exp) {
				t.Errorf("Expected HTML to contain %q, but got %q", exp, html)
			}
		}
	})

	t.Run("Variadic Class", func(t *testing.T) {
		el := dom.Div().Class("cls1", "cls2", "cls3")
		html := el.RenderHTML()
		if !fmt.Contains(html, "class='cls1 cls2 cls3'") {
			t.Errorf("Expected HTML to contain class='cls1 cls2 cls3', but got %q", html)
		}
	})
}
