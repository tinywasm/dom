package dom

import (
	"testing"

	"github.com/tinywasm/fmt"
)

func TestClass(t *testing.T) {
	attr := Class("btn-primary")
	if attr.Key != "class" || attr.Value != "btn-primary" {
		t.Errorf("Expected class='btn-primary', got %s='%s'", attr.Key, attr.Value)
	}
}

func TestClasses(t *testing.T) {
	attr := Classes("btn", "btn-primary")
	if attr.Key != "class" || attr.Value != "btn btn-primary" {
		t.Errorf("Expected class='btn btn-primary', got %s='%s'", attr.Key, attr.Value)
	}
}

func TestElementAddAttr(t *testing.T) {
	el := Div(Class("my-class"), fmt.KeyValue{Key: "data-test", Value: "val"})
	html := elementToHTML(el)
	expected := "<div class='my-class' data-test='val'></div>"
	if html != expected {
		t.Errorf("Expected %s, got %s", expected, html)
	}
}

type mockCSSProvider struct{}

func (m *mockCSSProvider) RenderCSS() any {
	return nil
}
