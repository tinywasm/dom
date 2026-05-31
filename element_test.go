package dom

import (
	"fmt"
	"strings"
	"testing"
)

func TestElement_ImplementsStringer(t *testing.T) {
	var _ fmt.Stringer = &Element{} // compile-time check
}

func TestElement_String_Basic(t *testing.T) {
	el := &Element{tag: "div"}
	el.Class("root").Text("hello")
	got := el.String()
	if !strings.Contains(got, "class='root'") {
		t.Error("expected class")
	}
	if !strings.Contains(got, "hello") {
		t.Error("expected text")
	}
}
