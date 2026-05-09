//go:build wasm

package dom

// SetTheme sets the data-theme attribute on the <html> element.
// If ThemeAuto is passed, the attribute is removed.
func SetTheme(theme Theme) {
	d := instance.(*domWasm)
	html := d.document.Get("documentElement")
	if theme == ThemeAuto {
		html.Call("removeAttribute", "data-theme")
		return
	}
	html.Call("setAttribute", "data-theme", string(theme))
}

// GetTheme returns the current value of the data-theme attribute on the <html> element.
// Returns ThemeAuto if the attribute is missing or empty.
func GetTheme() Theme {
	d := instance.(*domWasm)
	html := d.document.Get("documentElement")
	val := html.Call("getAttribute", "data-theme")
	if val.IsNull() || val.IsUndefined() {
		return ThemeAuto
	}
	t := Theme(val.String())
	if t == "" {
		return ThemeAuto
	}
	return t
}
