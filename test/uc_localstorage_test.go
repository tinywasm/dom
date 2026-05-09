//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

func TestLocalStorage_SetGet_Roundtrip(t *testing.T) {
	dom.LocalStorageClear()
	key := "test_key"
	val := "test_value"
	dom.LocalStorageSet(key, val)
	got := dom.LocalStorageGet(key)
	if got != val {
		t.Errorf("expected %s, got %s", val, got)
	}
}

func TestLocalStorage_Get_MissingKey_ReturnsEmpty(t *testing.T) {
	dom.LocalStorageClear()
	got := dom.LocalStorageGet("non_existent")
	if got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

func TestLocalStorage_Del_RemovesKey(t *testing.T) {
	dom.LocalStorageClear()
	key := "test_key"
	dom.LocalStorageSet(key, "val")
	dom.LocalStorageDel(key)
	got := dom.LocalStorageGet(key)
	if got != "" {
		t.Errorf("expected empty string after deletion, got %s", got)
	}
}

func TestLocalStorage_Set_Overwrites(t *testing.T) {
	dom.LocalStorageClear()
	key := "test_key"
	dom.LocalStorageSet(key, "val1")
	dom.LocalStorageSet(key, "val2")
	got := dom.LocalStorageGet(key)
	if got != "val2" {
		t.Errorf("expected val2, got %s", got)
	}
}

func TestLocalStorage_Clear_RemovesAll(t *testing.T) {
	dom.LocalStorageClear()
	dom.LocalStorageSet("k1", "v1")
	dom.LocalStorageSet("k2", "v2")
	dom.LocalStorageClear()
	if dom.LocalStorageGet("k1") != "" || dom.LocalStorageGet("k2") != "" {
		t.Error("expected all keys to be cleared")
	}
}

func TestLocalStorage_SetEmptyValue(t *testing.T) {
	dom.LocalStorageClear()
	dom.LocalStorageSet("key", "")
	got := dom.LocalStorageGet("key")
	if got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}
