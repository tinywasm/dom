//go:build !wasm

package dom

// LocalStorageGet is a stub for non-WASM environments.
func LocalStorageGet(key string) string {
	return ""
}

// LocalStorageSet is a stub for non-WASM environments.
func LocalStorageSet(key, value string) {}

// LocalStorageDel is a stub for non-WASM environments.
func LocalStorageDel(key string) {}

// LocalStorageClear is a stub for non-WASM environments.
func LocalStorageClear() {}
