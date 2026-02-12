//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

func TestFormAPI(t *testing.T) {
	_ = SetupDOM(t)

	t.Run("Void Elements HTML", func(t *testing.T) {
		tests := []struct {
			el       *dom.Element
			expected string
		}{
			{dom.Br(), "<br>"},
			{dom.Hr(), "<hr>"},
			{dom.Img("src.png", "alt text"), "<img src='src.png' alt='alt text'>"},
		}

		for _, tc := range tests {
			html := tc.el.RenderHTML()
			if html != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, html)
			}
		}
	})

	t.Run("Input Factories", func(t *testing.T) {
		tests := []struct {
			el       dom.Component
			expected string
		}{
			{dom.Text("email", "Your Email"), "<input type='text' name='email' placeholder='Your Email'>"},
			{dom.Email("user_email"), "<input type='email' name='user_email'>"},
			{dom.Password("pwd"), "<input type='password' name='pwd'>"},
			{dom.Checkbox("agree", "yes").Checked(), "<input type='checkbox' name='agree' value='yes' checked=''>"},
		}

		for _, tc := range tests {
			html := tc.el.RenderHTML()
			if !fmt.Contains(html, tc.expected) {
				t.Errorf("Expected HTML to contain %q, but got %q", tc.expected, html)
			}
		}
	})

	t.Run("JSX-like Containers", func(t *testing.T) {
		el := dom.Div(
			dom.H1("Title"),
			dom.P("Description"),
			dom.Form(
				dom.Text("q"),
				dom.Button("Search"),
			).Action("/search"),
		)

		html := el.RenderHTML()
		expected := []string{
			"<h1>Title</h1>",
			"<p>Description</p>",
			"<form action='/search'",
			"<input type='text' name='q'>",
			"<button>Search</button>",
		}

		for _, exp := range expected {
			if !fmt.Contains(html, exp) {
				t.Errorf("Expected HTML to contain %q, but got %q", exp, html)
			}
		}
	})

	t.Run("Select and Options", func(t *testing.T) {
		el := dom.Select("role",
			dom.Option("admin", "Admin"),
			dom.SelectedOption("user", "User"),
		)
		html := el.RenderHTML()
		expected := []string{
			"<select name='role'>",
			"<option value='admin'>Admin</option>",
			"<option value='user' selected=''>User</option>",
		}
		for _, exp := range expected {
			if !fmt.Contains(html, exp) {
				t.Errorf("Expected HTML to contain %q, but got %q", exp, html)
			}
		}
	})

	t.Run("Event Wiring on Specialized Elements", func(t *testing.T) {
		triggered := false
		input := dom.Text("email").On("input", func(e dom.Event) {
			triggered = true
		})

		// Use Render to trigger event wiring
		dom.Render("root", input)

		// Simulate event
		TriggerEvent(input.GetID(), "input", "")

		if !triggered {
			t.Error("Event on specialized element was not triggered")
		}
	})
}

func TriggerEvent(id, eventType string, value string) {
	doc := js.Global().Get("document")
	rawEl := doc.Call("getElementById", id)
	if !rawEl.IsNull() && !rawEl.IsUndefined() {
		if value != "" {
			rawEl.Set("value", value)
		}
		event := js.Global().Get("Event").New(eventType, map[string]interface{}{
			"bubbles": true,
		})
		rawEl.Call("dispatchEvent", event)
	}
}
