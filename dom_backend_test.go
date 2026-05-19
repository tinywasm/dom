//go:build !wasm

package dom

import (
	"testing"
)

func TestBackendStubs(t *testing.T) {
	td := &tinyDOM{}
	d := newDom(td)

	if _, ok := d.(interface {
		Get(string) (Reference, bool)
	}).Get("test"); !ok {
		t.Error("get should return true (stub) on backend")
	}

	if err := d.Render("p", nil); err == nil {
		t.Error("Render should return error on backend")
	}

	if err := d.Append("p", nil); err == nil {
		t.Error("Append should return error on backend")
	}

	d.Update(nil)

	d.(interface{ unmount(Component) }).unmount(nil)
	d.OnHashChange(func(h string) {})
	if d.GetHash() != "" {
		t.Error("GetHash should return empty string on backend")
	}
	d.SetHash("test")
}
