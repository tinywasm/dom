# PLAN: Remove Form Functions from dom

**Module:** `github.com/tinywasm/dom`
**Breaking change:** Yes — removes form-related functions, types, and files.
**Status:** Pending
**Reason:** Form elements with validation live in `tinywasm/form`. dom should only provide pure HTML layout elements. One way to create forms: `form.New()` (schema-driven).

---

## What to Remove

### Files to DELETE

| File | Contents |
|---|---|
| input.go | `InputEl` struct, `Input()`, `Text()`, `Email()`, `Password()`, `Number()`, `Checkbox()`, `Radio()`, `File()`, `Date()`, `Hidden()`, `Search()`, `Tel()`, `Url()`, `Range()`, `Color()`, `Submit()`, `Reset()` |
| form_el.go | `FormEl` struct, `Form()` |
| select_el.go | `SelectEl` struct, `Select()` |
| textarea_el.go | `TextareaEl` struct, `Textarea()` |

### What STAYS in dom

- All layout elements: Div, P, H1-H6, Nav, Section, Header, Footer, Main, etc.
- Generic elements used in forms: Button, Label, Fieldset, Legend
- Option, SelectedOption (simple element helpers)
- Core: Element, Component interface, Render, Append, Update, Get
- Events, lifecycle hooks, Reference

---

## Files to Update

### Code files

- [x] **Delete** input.go, form_el.go, select_el.go, textarea_el.go
- [x] **Delete** test/uc_form_api_test.go (tests removed functions)
- [x] **Update** test/uc_coverage_test.go — remove form-related test cases
- [x] **Update** web/client.go — remove form usage from demo app (or update to use tinywasm/form)

### External consumers

- [ ] **Update** components/input/input.go — uses `dom.Input()`. This component likely needs to be reworked or deleted if form/input replaces it
- [ ] **Update** components/form/form.go — uses `dom.Form()`. Same: rework or delete if form.New() replaces it

### Documentation

- [x] **Update** README.md — remove form examples, document that forms live in tinywasm/form
- [x] **Update** docs/ARCHITECTURE.md — remove FormEl, InputEl references

---

## Verification

1. `gotest ./...` — all remaining tests pass
2. `grep -r "InputEl\|FormEl\|SelectEl\|TextareaEl" .` — no references
3. `grep -r "dom.Email\|dom.Form\|dom.Password\|dom.Input\|dom.Select\|dom.Textarea" ../` — no external references
