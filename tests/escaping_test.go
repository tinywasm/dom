package dom_test

import (
	"testing"

	. "github.com/tinywasm/dom"
)

type dummyComp struct {
	Element
	id string
}

func (c *dummyComp) GetID() string { return c.id }

func (c *dummyComp) SetID(id string) { c.id = id }

func (c *dummyComp) Render() *Element {
	return NewElement("span").ID(c.id).Text("comp-content")
}

func (c *dummyComp) String() string {
	return c.Render().String()
}

func containsSub(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestTextEscapesMarkup(t *testing.T) {
	el := NewElement("div").Text("<img src=x onerror=alert(1)>")
	got := el.String()
	if containsSub(got, "<img") {
		t.Fatalf("expected markup to be escaped, but found raw '<img' in: %s", got)
	}
	if !containsSub(got, "&lt;img") {
		t.Fatalf("expected '&lt;img' in output, got: %s", got)
	}
}

func TestTextEscapesAllFive(t *testing.T) {
	el := NewElement("div").Text(`& < > " '`)
	got := el.String()
	want := "<div>&amp; &lt; &gt; &quot; &#39;</div>"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestTextEscapesAmpersandFirst(t *testing.T) {
	el := NewElement("div").Text("&lt;")
	got := el.String()
	want := "<div>&amp;lt;</div>"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestRawPassesThrough(t *testing.T) {
	el := NewElement("div").Raw(Trust("<b>x</b>"))
	got := el.String()
	want := "<div><b>x</b></div>"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestBindTextEscapes(t *testing.T) {
	sig := NewString("<script>alert(1)</script>")
	el := NewElement("div").BindText(sig)
	got := el.String()
	if containsSub(got, "<script>") {
		t.Fatalf("expected BindText value to be escaped, but found '<script>' in: %s", got)
	}
	if !containsSub(got, "&lt;script&gt;") {
		t.Fatalf("expected '&lt;script&gt;' in output, got: %s", got)
	}
}

func TestBindTextFuncEscapes(t *testing.T) {
	el := NewElement("div").BindTextFunc(func() string {
		return "<script>alert(1)</script>"
	})
	got := el.String()
	if containsSub(got, "<script>") {
		t.Fatalf("expected BindTextFunc value to be escaped, but found '<script>' in: %s", got)
	}
	if !containsSub(got, "&lt;script&gt;") {
		t.Fatalf("expected '&lt;script&gt;' in output, got: %s", got)
	}
}

func TestIDIsEscaped(t *testing.T) {
	el := NewElement("div").ID("a' onload='alert(1)")
	got := el.String()
	if containsSub(got, "a' onload") {
		t.Fatalf("expected ID to be attribute escaped, but found raw quote in: %s", got)
	}
	if !containsSub(got, "&#39;") {
		t.Fatalf("expected '&#39;' in escaped ID, got: %s", got)
	}
}

func TestAttrKeyIsEscaped(t *testing.T) {
	el := NewElement("div").Attr("a' onload='x", "v")
	got := el.String()
	if containsSub(got, "a' onload") {
		t.Fatalf("expected attribute key to be escaped, but found unescaped quote in: %s", got)
	}
	if !containsSub(got, "a&#39; onload=&#39;x") {
		t.Fatalf("expected escaped attribute key, got: %s", got)
	}
}

func TestSingleSerializer(t *testing.T) {
	comp := &dummyComp{id: "sub-1"}
	el := NewElement("div").
		ID("root").
		Text("<header>").
		Raw(Trust("<b>raw</b>")).
		Child(comp)

	got := el.String()
	want := "<div id='root'>&lt;header&gt;<b>raw</b><span id='sub-1'>comp-content</span></div>"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestAdminTable_RendersHostileUserDataInert(t *testing.T) {
	hostileInputs := []string{
		`<img src=x onerror=alert(1)>`,
		`"><script>alert(1)</script>`,
		`javascript:alert(1)`,
	}

	table := NewElement("table")
	tbody := NewElement("tbody")
	for _, hostile := range hostileInputs {
		row := NewElement("tr").Child(
			NewElement("td").Text(hostile),
		)
		tbody.Child(row)
	}
	table.Child(tbody)

	rendered := table.String()

	forbidden := []string{"<img", "<script"}
	for _, f := range forbidden {
		if containsSub(rendered, f) {
			t.Errorf("hostile data escaped check failed: found forbidden %q in serialized table:\n%s", f, rendered)
		}
	}
}
