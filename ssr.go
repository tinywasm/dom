//go:build !wasm

package dom

func (c CssVars) RenderCSS() string {
	return c.renderCSS()
}
