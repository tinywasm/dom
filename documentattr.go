//go:build wasm

package dom

// SetDocumentAttr sets an attribute on document.documentElement (<html>).
// value=="" removes the attribute — consistent with GetDocumentAttr returning ""
// for absent attributes.
func SetDocumentAttr(attr, value string) {
	html := instance.(*domWasm).document.Get("documentElement")
	if !html.Truthy() {
		return
	}
	if value == "" {
		html.Call("removeAttribute", attr)
	} else {
		html.Call("setAttribute", attr, value)
	}
}

// GetDocumentAttr reads an attribute from document.documentElement.
// Returns "" if the attribute is absent.
func GetDocumentAttr(attr string) string {
	html := instance.(*domWasm).document.Get("documentElement")
	if !html.Truthy() {
		return ""
	}
	v := html.Call("getAttribute", attr)
	if v.IsNull() || v.IsUndefined() {
		return ""
	}
	return v.String()
}
