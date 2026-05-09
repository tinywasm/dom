//go:build !wasm

package dom

// SetDocumentAttr is a no-op on the backend.
func SetDocumentAttr(_, _ string) {}

// GetDocumentAttr returns an empty string on the backend.
func GetDocumentAttr(_ string) string { return "" }
