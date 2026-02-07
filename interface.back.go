//go:build !wasm

package dom

// CSSProvider is an optional interface for components that need to inject CSS.
type CSSProvider interface {
	RenderCSS() string
}

// JSProvider is an optional interface for components that need to inject JS.
type JSProvider interface {
	RenderJS() string
}

// IconSvgProvider is an optional interface for components that provide SVG icons.
type IconSvgProvider interface {
	IconSvg() map[string]string
}
