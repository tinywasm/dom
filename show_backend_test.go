//go:build !wasm

package dom

import (
	"strings"
	"testing"
)

func TestShowBackend(t *testing.T) {
	off := Show(NewBool(false), NewElement("span").Text("x"))
	html := off.String()
	if !strings.Contains(html, "display:none") {
		t.Error("hidden Show must carry display:none in SSR markup")
	}
	if !strings.Contains(html, "<span") {
		t.Error("child must be serialized even while hidden (WASM parity)")
	}

	on := Show(NewBool(true), NewElement("span").Text("x"))
	if strings.Contains(on.String(), "display:none") {
		t.Error("visible Show must not carry display:none")
	}
}
