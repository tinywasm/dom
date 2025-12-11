//go:build !wasm

package dom

// CSSRenderer is an optional interface for components that need to inject CSS.
// Only used in backend SSR.
type CSSRenderer interface {
	Component
	RenderCSS() string
}

// JSRenderer is an optional interface for components that need to inject JS.
// Only used in backend SSR.
type JSRenderer interface {
	Component
	RenderJS() string
}
