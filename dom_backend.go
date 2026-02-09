//go:build !wasm

package dom

import "github.com/tinywasm/fmt"

// domBackend is a stub implementation for non-WASM environments (e.g., SSR).
type domBackend struct {
	*tinyDOM
}

// newDom returns a new instance of the domBackend.
func newDom(td *tinyDOM) DOM {
	return &domBackend{
		tinyDOM: td,
	}
}

// Get is not implemented for backend.
func (d *domBackend) Get(id string) (Element, bool) {
	return nil, false
}

// Render is not implemented for backend, but we provide a helper to render a component to HTML string.
func (d *domBackend) Render(parentID string, component Component) error {
	// In SSR, Render usually just returns the HTML to be sent to the client.
	// Since DOM interface expects a side effect on a parent, and there is no real DOM,
	// we just return an error or log it.
	// However, for consistency, we should allow it to work if we want to "simulated" render.
	return fmt.Err("Render to parent is not supported on backend. Use RenderHTML() instead.")
}

func (d *domBackend) renderToHTML(n Node) string {
	s := "<" + n.Tag
	for _, attr := range n.Attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
	s += ">"
	for _, child := range n.Children {
		switch v := child.(type) {
		case Node:
			s += d.renderToHTML(v)
		case string:
			s += v
		case Component:
			s += v.RenderHTML()
		}
	}
	s += "</" + n.Tag + ">"
	return s
}

func (d *domBackend) Append(parentID string, component Component) error {
	return fmt.Err("Append not supported in backend/stub")
}

// Hydrate is not implemented for backend.
func (d *domBackend) Hydrate(parentID string, component Component) error {
	return fmt.Err("Hydrate is not implemented for backend")
}

// Update is not implemented for backend.
func (d *domBackend) Update(component Component) error {
	return fmt.Err("Update is not implemented for backend")
}

// Unmount is not implemented for backend.
func (d *domBackend) Unmount(component Component) {
}

func (d *domBackend) OnHashChange(handler func(hash string)) {}
func (d *domBackend) GetHash() string                        { return "" }
func (d *domBackend) SetHash(hash string)                    {}
func (d *domBackend) QueryAll(selector string) []Element     { return nil }
