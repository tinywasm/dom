---
PLAN: "fix: escape HTML attribute and class values in elementToHTML/renderToHTML"
REVIEWER: none
STATUS: review
SESSION: 17815455470289849777
PR: https://github.com/tinywasm/dom/pull/21
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — escape attribute/class values in HTML serialization

## Prerequisite

Install the test runner first — it is not globally available in your environment:

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

Use `gotest` (not `go test`) for every test run in this plan.

## Problem

Two functions in this repo serialize a `dom.Element` tree to an HTML string by
string concatenation, wrapping every attribute value and class token in
**raw, unescaped single quotes**:

- `elementToHTML` in `element.go` (the `!wasm`/SSR path)
- `domWasm.renderToHTML` in `dom_frontend.go` (the `wasm` runtime path — it
  builds the same kind of HTML string and injects it via `innerHTML`, so it
  has the identical bug)

If an attribute value itself contains a raw `'` character, the HTML parser
reads that `'` as the *closing* quote of the attribute, and everything after
it becomes garbage. Confirmed live: an `<img>` with
`src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' ..."` gets
truncated to `src="data:image/svg+xml,%3Csvg xmlns="` — the image never
loads, in both the SSR output and the live WASM DOM (both paths funnel
through this same string-building code and are injected via `innerHTML`).

The fix: escape every attribute value and class token before writing it
between the quotes. Do NOT change the quote character (stays `'`) and do NOT
touch text-content serialization (the `hasTextContent`/children branches in
either function) — that is a separate concern, out of scope here.

## Fix

This repo already depends on `github.com/tinywasm/fmt` (see `go.mod`), which
already ships `(*Conv) EscapeAttr() string` in `fmt/html.go` — HTML-entity
escaping (`&`, `"`, `'`, `<`, `>`) purpose-built for exactly this: making a
string safe inside a single-quoted (or double-quoted) HTML attribute. Use it
via `fmt.Convert(value).EscapeAttr()`. Both files already import
`"github.com/tinywasm/fmt"` as `fmt` — no new import needed in either.

Do NOT use `encoding/html` (stdlib) or write a new escaping helper — this
project forbids stdlib in WASM-compiled code (`dom_frontend.go` carries
`//go:build wasm`), and `fmt.Convert(...).EscapeAttr()` already covers both
build tags correctly since `tinywasm/fmt` itself has no stdlib dependency in
its WASM path.

### 1. `element.go`

Find this exact block (around line 367-379):

```go
	if len(classes) > 0 {
		s += " class='"
		for i, c := range classes {
			if i > 0 {
				s += " "
			}
			s += c
		}
		s += "'"
	}
	for _, attr := range attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
```

Replace it with:

```go
	if len(classes) > 0 {
		s += " class='"
		for i, c := range classes {
			if i > 0 {
				s += " "
			}
			s += fmt.Convert(c).EscapeAttr()
		}
		s += "'"
	}
	for _, attr := range attrs {
		s += " " + attr.Key + "='" + fmt.Convert(attr.Value).EscapeAttr() + "'"
	}
```

Only the value (`c`, `attr.Value`) is escaped. `attr.Key` is never
user-controlled (it comes from internal typed attribute constants) — leave it
untouched.

### 2. `dom_frontend.go`

Find the identical block (around line 578-590, inside `renderToHTML`):

```go
	if len(classes) > 0 {
		s += " class='"
		for i, c := range classes {
			if i > 0 {
				s += " "
			}
			s += c
		}
		s += "'"
	}
	for _, attr := range attrs {
		s += " " + attr.Key + "='" + attr.Value + "'"
	}
```

Apply the exact same replacement as in step 1 (same before/after code).

## Tests

### 3. `dom_test.go` (SSR path — `!wasm`)

Add this test function (it already imports `"github.com/tinywasm/fmt"`, no
import change needed):

```go
func TestElementAttrEscaping(t *testing.T) {
	el := (&Element{tag: "img"}).Set(fmt.KeyValue{Key: "src", Value: "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg'%3E%3C/svg%3E"})
	html := elementToHTML(el)
	expected := "<img src='data:image/svg+xml,%3Csvg xmlns=&#39;http://www.w3.org/2000/svg&#39;%3E%3C/svg%3E'></img>"
	if html != expected {
		t.Errorf("Expected %s, got %s", expected, html)
	}
}
```

### 4. `dom_internal_test.go` (WASM path — `//go:build wasm`)

This file currently imports only `"syscall/js"` and `"testing"`. Add
`"github.com/tinywasm/fmt"` to its import block, then add this subtest inside
the existing `t.Run("renderToHTML", ...)` block (after the two existing
sub-cases in that block, before its closing `})`):

```go
		// Regression: a raw single quote inside an attribute value must not
		// truncate the attribute — see docs — the value must come out
		// HTML-entity-escaped.
		elEsc := (&Element{tag: "img"}).Set(fmt.KeyValue{Key: "src", Value: "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg'%3E%3C/svg%3E"})
		var compsEsc []Component
		htmlEsc := d.renderToHTML(elEsc, &compsEsc, "parent-id")
		expectedEsc := "<img src='data:image/svg+xml,%3Csvg xmlns=&#39;http://www.w3.org/2000/svg&#39;%3E%3C/svg%3E'></img>"
		if htmlEsc != expectedEsc {
			t.Errorf("expected %q, got %q", expectedEsc, htmlEsc)
		}
```

## Acceptance Criteria

- `gotest` (full suite, from the repo root) passes — this covers both the
  `!wasm` and `wasm` build tags automatically.
- `grep -n "\"'\" + attr.Value + \"'\"" element.go dom_frontend.go` and
  `grep -n "s += c$" element.go dom_frontend.go` both return **no matches**
  (confirms the raw, unescaped concatenation is gone from both files).
- The two new tests (`TestElementAttrEscaping` in `dom_test.go`, and the new
  subtest in `dom_internal_test.go`) exist and pass.
- No other files are touched. No stdlib import (`encoding/html`, `strings`,
  etc.) is added anywhere.

## Stages

| # | Stage | Files |
|---|-------|-------|
| 1 | Escape SSR attribute/class serialization | `element.go` |
| 2 | Escape WASM attribute/class serialization | `dom_frontend.go` |
| 3 | Add SSR regression test | `dom_test.go` |
| 4 | Add WASM regression test | `dom_internal_test.go` |
| 5 | Run `gotest`, verify acceptance criteria | — |
