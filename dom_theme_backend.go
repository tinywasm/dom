//go:build !wasm

package dom

// SetTheme is a stub for non-WASM environments.
func SetTheme(theme Theme) {}

// GetTheme is a stub for non-WASM environments.
func GetTheme() Theme {
	return ThemeAuto
}
