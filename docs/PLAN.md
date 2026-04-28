# PLAN: Preserve Active Element Focus and Cursor During Update()

## Bug Description

When `Update()` is called on a component while the user is typing in a child input element, the cursor jumps to position 0 on every keystroke.

**Observed in:** `selectsearch` — every character typed in the search input resets the cursor to the beginning.

---

## Root Cause (two layers)

### Layer 1 — `dom_frontend.go`: `outerHTML` destroys active element

`Update()` replaces the component via `outerHTML`, which removes the focused input from the DOM. The browser moves focus to `document.body`. `dom_frontend.go` already snapshots the active element ID and calls `focus()` on the restored element — but `focus()` by HTML spec always places the cursor at **position 0** when called programmatically. It does not preserve cursor position.

This is not a browser quirk: the same happens in plain JavaScript. Any library that does full element replacement must explicitly restore cursor via `setSelectionRange` after `focus()`.

### Layer 2 — `selectsearch/front.go`: redundant `Focus()` call

The `input` event handler calls `Focus()` explicitly after `c.Update()`. This second `focus()` call overwrites whatever cursor position `dom_frontend.go` sets, compounding the problem.

---

## Fix Plan

### Step 1 — `dom_frontend.go`: snapshot cursor, restore with `setSelectionRange`

```go
// Before outerHTML:
activeEl := d.document.Get("activeElement")
activeID := ""
cursorStart, cursorEnd := 0, 0
if !activeEl.IsNull() && !activeEl.IsUndefined() {
    activeID = activeEl.Get("id").String()
    cs := activeEl.Get("selectionStart")
    ce := activeEl.Get("selectionEnd")
    if !cs.IsNull() && !cs.IsUndefined() {
        cursorStart = cs.Int()
    }
    if !ce.IsNull() && !ce.IsUndefined() {
        cursorEnd = ce.Int()
    }
}

// After outerHTML + OnMount rewiring:
if activeID != "" {
    restored := d.document.Call("getElementById", activeID)
    if !restored.IsNull() && !restored.IsUndefined() {
        restored.Call("focus")
        cs := restored.Get("selectionStart")
        if !cs.IsNull() && !cs.IsUndefined() {
            restored.Call("setSelectionRange", cursorStart, cursorEnd)
        }
    }
}
```

> `selectionStart` is `null` for non-text inputs (checkboxes, buttons) — the null guard prevents errors.

### Step 2 — `selectsearch/front.go`: remove redundant `Focus()`

Once `dom.Update()` handles focus + cursor restoration automatically, the explicit `Focus()` call after `c.Update()` in the `input` handler must be removed. Keeping it would overwrite the correctly restored cursor position.

---

## Test

**File:** `dom/test/uc_focus_preserve_test.go`
**Test:** `TestUpdate_PreservesActiveElementFocus`

Verifies that after `c.Update()` triggered by an `input` event:
1. `document.activeElement.id` equals the input's ID (focus preserved).
2. `selectionStart == 2` (cursor at position 2, not reset to 0).

Currently **FAILS** at step 2: `selectionStart=0, want 2`.

Run:
```bash
gotest -run TestUpdate_PreservesActiveElementFocus
```
