package dom

import (
	"testing"

	"github.com/tinywasm/css"
	"github.com/tinywasm/fmt"
)

func TestElementAddAttr(t *testing.T) {
	cls := css.Class("my-class")
	el := (&Element{tag: "div"}).Add(cls.AsAttr(), fmt.KeyValue{Key: "data-test", Value: "val"})
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
