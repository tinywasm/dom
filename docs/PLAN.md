# PLAN: Add CssVars + theme.css to tinywasm/dom

## Context

`tinywasm/dom` is the foundational UI package for the tinywasm ecosystem. It defines
the `Component`, `CSSProvider`, `ViewRenderer` and related interfaces used by all
components and layouts.

Currently there is no unified CSS token standard across the ecosystem. Components use
inconsistent naming (`--color-primary`, `--col-pri`, `--primary`). This plan adds:

1. `CssVars` struct — Go type that generates a `:root { }` CSS block with the standard
   token set, dark-mode variants included.
2. `theme.css` — canonical CSS file consumed by `CSSProvider` in `ssr.go` (build tag
   `!wasm`), defining the default light theme, automatic dark mode via
   `@media (prefers-color-scheme: dark)`, and manual override via `[data-theme]`.

This makes `tinywasm/dom` the single source of truth for design tokens. Every package
that already depends on `dom` gets the standard at zero extra cost.

**Must be executed BEFORE `tinywasm/layout` PLAN.**

---

## Standard token names (canonical list)

| Go field          | CSS variable          | Semantic role                        |
|-------------------|-----------------------|--------------------------------------|
| Primary           | --color-primary       | Main text / foreground               |
| Secondary         | --color-secondary     | Accent / brand color                 |
| Tertiary          | --color-tertiary      | Muted text, borders                  |
| Quaternary        | --color-quaternary    | Deep background / shadows            |
| Gray              | --color-gray          | Neutral surface                      |
| Selection         | --color-selection     | Hover / selected state               |
| Hover             | --color-hover         | Hover accent                         |
| Success           | --color-success       | Success feedback                     |
| Error             | --color-error         | Error / danger feedback              |
| MenuWidthCollapsed| --menu-width-collapsed| Collapsed navigation width           |
| MenuWidthExpanded | --menu-width-expanded | Expanded navigation width            |
| TitleHeight       | --title-height        | Module title bar height              |
| ContentHeight     | --content-height      | Module content area height           |
| ControlsHeight    | --controls-height     | Controls bar height                  |
| SpacingPrimary    | --mag-pri             | Primary spacing unit                 |
| SpacingSecondary  | --mag-sec             | Secondary spacing unit               |
| SpacingQuaternary | --mag-cua             | Quaternary spacing unit              |

---

## Files to create / modify

### 1. `theme.go` (new, build tag `!wasm` — CSS string generation for SSR only)

```go
//go:build !wasm

package dom

import "github.com/tinywasm/fmt"

// CssVars defines the design token set for a tinywasm project.
// Each field maps to a CSS custom property via the `css` struct tag.
// Use Render() to generate the :root { } block for SSR injection.
type CssVars struct {
    Primary             string `css:"--color-primary"`
    Secondary           string `css:"--color-secondary"`
    Tertiary            string `css:"--color-tertiary"`
    Quaternary          string `css:"--color-quaternary"`
    Gray                string `css:"--color-gray"`
    Selection           string `css:"--color-selection"`
    Hover               string `css:"--color-hover"`
    Success             string `css:"--color-success"`
    Error               string `css:"--color-error"`
    MenuWidthCollapsed  string `css:"--menu-width-collapsed"`
    MenuWidthExpanded   string `css:"--menu-width-expanded"`
    TitleHeight         string `css:"--title-height"`
    ContentHeight       string `css:"--content-height"`
    ControlsHeight      string `css:"--controls-height"`
    SpacingPrimary      string `css:"--mag-pri"`
    SpacingSecondary    string `css:"--mag-sec"`
    SpacingQuaternary   string `css:"--mag-cua"`
}

// DefaultCssVars returns the default light-theme token set.
func DefaultCssVars() CssVars {
    return CssVars{
        Primary:            "#ffffff",
        Secondary:          "#7c3aed",
        Tertiary:           "#94a3b8",
        Quaternary:         "#1e293b",
        Gray:               "#f8fafc",
        Selection:          "#a78bfa",
        Hover:              "#6d28d9",
        Success:            "#10b981",
        Error:              "#ef4444",
        MenuWidthCollapsed: "64px",
        MenuWidthExpanded:  "250px",
        TitleHeight:        "8vh",
        ContentHeight:      "89vh",
        ControlsHeight:     "3vh",
        SpacingPrimary:     "0.5rem",
        SpacingSecondary:   "0.2rem",
        SpacingQuaternary:  "0.2rem",
    }
}

// Render returns the CSS :root { } block with all non-empty tokens.
// Safe to call from SSR. Does NOT use reflect — iterates fields explicitly.
func (c CssVars) Render() string {
    type kv struct{ k, v string }
    pairs := []kv{
        {"--color-primary", c.Primary},
        {"--color-secondary", c.Secondary},
        {"--color-tertiary", c.Tertiary},
        {"--color-quaternary", c.Quaternary},
        {"--color-gray", c.Gray},
        {"--color-selection", c.Selection},
        {"--color-hover", c.Hover},
        {"--color-success", c.Success},
        {"--color-error", c.Error},
        {"--menu-width-collapsed", c.MenuWidthCollapsed},
        {"--menu-width-expanded", c.MenuWidthExpanded},
        {"--title-height", c.TitleHeight},
        {"--content-height", c.ContentHeight},
        {"--controls-height", c.ControlsHeight},
        {"--mag-pri", c.SpacingPrimary},
        {"--mag-sec", c.SpacingSecondary},
        {"--mag-cua", c.SpacingQuaternary},
    }
    sb := fmt.GetConv()
    sb.WriteString(":root {\n")
    for _, p := range pairs {
        if p.v != "" {
            sb.WriteString("  " + p.k + ": " + p.v + ";\n")
        }
    }
    sb.WriteString("}\n")
    return sb.String()
}
```

**Key decision:** No `reflect`. Explicit pairs slice — consistent with the ecosystem
pattern of avoiding reflect for runtime code (ormc uses it at code-gen time only).

---

### 2. `theme.css` (new, embedded in `theme_ssr.go`)

```css
/*
 * tinywasm/dom — canonical design tokens
 * Light theme (default). Dark via prefers-color-scheme. Manual via [data-theme].
 */

/* ── Light (default) ─────────────────────────────────────────── */
:root {
  --color-primary:    #ffffff;
  --color-secondary:  #7c3aed;
  --color-tertiary:   #94a3b8;
  --color-quaternary: #1e293b;
  --color-gray:       #f8fafc;
  --color-selection:  #a78bfa;
  --color-hover:      #6d28d9;
  --color-success:    #10b981;
  --color-error:      #ef4444;

  --menu-width-collapsed: 64px;
  --menu-width-expanded:  250px;

  --title-height:    8vh;
  --content-height:  89vh;
  --controls-height: 3vh;

  --mag-pri: 0.5rem;
  --mag-sec: 0.2rem;
  --mag-cua: 0.2rem;
}

/* ── Automatic dark (OS preference, no JS required) ──────────── */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --color-primary:    #e2e8f0;
    --color-secondary:  #7c3aed;
    --color-tertiary:   #475569;
    --color-quaternary: #f1f5f9;
    --color-gray:       #0f172a;
    --color-selection:  #6d28d9;
    --color-hover:      #a78bfa;
  }
}

/* ── Manual dark override ([data-theme="dark"] on <html>) ────── */
[data-theme="dark"] {
  --color-primary:    #e2e8f0;
  --color-secondary:  #7c3aed;
  --color-tertiary:   #475569;
  --color-quaternary: #f1f5f9;
  --color-gray:       #0f172a;
  --color-selection:  #6d28d9;
  --color-hover:      #a78bfa;
}

/* ── Manual light override ───────────────────────────────────── */
[data-theme="light"] {
  --color-primary:    #ffffff;
  --color-secondary:  #7c3aed;
  --color-tertiary:   #94a3b8;
  --color-quaternary: #1e293b;
  --color-gray:       #f8fafc;
  --color-selection:  #a78bfa;
  --color-hover:      #6d28d9;
}
```

---

### 3. `theme_ssr.go` (new, build tag `!wasm` — embeds the CSS file)

```go
//go:build !wasm

package dom

import _ "embed"

//go:embed theme.css
var ThemeCSS string
```

`ThemeCSS` is a package-level string that `tinywasm/site` (or any SSR host) can inject
into the HTML `<head>` as a `<style>` block. It is NOT a `CSSProvider` itself — it is
the base theme, injected once by the site builder, not by individual components.

---

### 4. `theme_test.go` (new, build tag `!wasm`)

Test that:
- `DefaultCssVars().Render()` contains `:root {`
- All 17 tokens are present in the output
- An empty field is omitted from output
- `CssVars` with a custom `Primary` overrides only that token

---

## Dependencies

`theme.go` uses `github.com/tinywasm/fmt` (already a dependency of `tinywasm/dom`).
No new external dependencies.

## go.mod changes

None required — `tinywasm/fmt` is already in `go.mod`.

## Execution order

1. Create `theme.go`
2. Create `theme.css`
3. Create `theme_ssr.go`
4. Create `theme_test.go`
5. Run `go test ./...` — must pass
6. Commit: `feat: add CssVars and theme.css design token standard`

## Notes for Jules

- Do NOT use `reflect` in `theme.go`. The explicit pairs slice is intentional.
- `ThemeCSS` is a `string`, not a `CSSProvider`. The site builder injects it once.
- The `CssVars.Render()` output is used when a project wants to override tokens
  programmatically (e.g. per-tenant theming). The `theme.css` file is the static default.
- Dark mode uses `:root:not([data-theme="light"])` inside the media query so a user who
  explicitly chose light mode is never overridden by the OS dark preference.
- No JS is required for dark mode to work. The `[data-theme]` attribute is optional.
