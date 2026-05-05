//go:build !wasm

package dom

import _ "embed"

//go:embed theme.css
var rootCSS string

// RootCSS returns the default `:root { … }` theme as a CSS string.
// Static extractors (assetmin) read it via the `//go:embed` directive
// during AST walking. Apps that want a custom theme expose their own
// `RootCSS()` from the project root — assetmin resolves the override.
func RootCSS() string { return rootCSS }
