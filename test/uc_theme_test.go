//go:build wasm

package dom_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

func TestSetTheme_Dark_SetsAttribute(t *testing.T) {
	dom.SetTheme(dom.ThemeDark)
	html := js.Global().Get("document").Get("documentElement")
	got := html.Call("getAttribute", "data-theme").String()
	if got != "dark" {
		t.Errorf("expected dark, got %s", got)
	}
}

func TestSetTheme_Light_SetsAttribute(t *testing.T) {
	dom.SetTheme(dom.ThemeLight)
	html := js.Global().Get("document").Get("documentElement")
	got := html.Call("getAttribute", "data-theme").String()
	if got != "light" {
		t.Errorf("expected light, got %s", got)
	}
}

func TestSetTheme_Auto_RemovesAttribute(t *testing.T) {
	dom.SetTheme(dom.ThemeDark) // set first
	dom.SetTheme(dom.ThemeAuto)
	html := js.Global().Get("document").Get("documentElement")
	val := html.Call("getAttribute", "data-theme")
	if !val.IsNull() {
		t.Errorf("expected null, got %s", val.String())
	}
}

func TestSetTheme_PassesThrough_InvalidValue(t *testing.T) {
	dom.SetTheme(dom.Theme("xyz"))
	html := js.Global().Get("document").Get("documentElement")
	got := html.Call("getAttribute", "data-theme").String()
	if got != "xyz" {
		t.Errorf("expected xyz, got %s", got)
	}
}

func TestGetTheme_NoAttribute_ReturnsAuto(t *testing.T) {
	dom.SetTheme(dom.ThemeAuto) // remove attribute
	got := dom.GetTheme()
	if got != dom.ThemeAuto {
		t.Errorf("expected auto, got %s", got)
	}
}

func TestGetTheme_AfterSet_ReturnsValue(t *testing.T) {
	themes := []dom.Theme{dom.ThemeDark, dom.ThemeLight, dom.ThemeAuto}
	for _, want := range themes {
		dom.SetTheme(want)
		got := dom.GetTheme()
		if got != want {
			t.Errorf("expected %s, got %s", want, got)
		}
	}
}

func TestSetTheme_DoesNotTouchLocalStorage(t *testing.T) {
	dom.LocalStorageClear()
	dom.SetTheme(dom.ThemeDark)
	// We don't know what key it might use, but it shouldn't use any.
	// Check if any key exists.
	ls := js.Global().Get("localStorage")
	if ls.Get("length").Int() > 0 {
		t.Error("SetTheme should not touch localStorage")
	}
}
