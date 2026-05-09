//go:build wasm

package dom

import "syscall/js"

// LocalStorageGet retrieves a value from window.localStorage.
// Returns "" if the key does not exist OR if storage is unavailable.
func LocalStorageGet(key string) string {
	v := instance.(*domWasm).lsCall("getItem", key)
	if v.IsNull() || v.IsUndefined() {
		return ""
	}
	return v.String()
}

// LocalStorageSet sets a value in window.localStorage.
func LocalStorageSet(key, value string) {
	instance.(*domWasm).lsCall("setItem", key, value)
}

// LocalStorageDel removes a key from window.localStorage.
func LocalStorageDel(key string) {
	instance.(*domWasm).lsCall("removeItem", key)
}

// LocalStorageClear removes all keys from window.localStorage.
func LocalStorageClear() {
	instance.(*domWasm).lsCall("clear")
}

// lsCall — unique method over domWasm for localStorage. Centralizes:
//  1. The Truthy() guard (only possible defense in TinyGo WASM)
//  2. The Log of "unavailable"
func (d *domWasm) lsCall(method string, args ...any) js.Value {
	if !d.localStorage.Truthy() {
		d.Log("dom: localStorage unavailable, ignoring", method)
		return js.Value{}
	}
	return d.localStorage.Call(method, args...)
}
