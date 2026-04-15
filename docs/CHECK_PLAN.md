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

### 1. `ssr.theme.go` (new, build tag `!wasm` — CssVars + CSS embed, SSR only)

Both the `CssVars` type and the embedded `theme.css` live in the same file since both
are `!wasm` and belong to the same responsibility: providing theme tokens for SSR.

```go
//go:build !wasm

package dom

import (
    _ "embed"

    "github.com/tinywasm/fmt"
)

//go:embed theme.css
var ThemeCSS string

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

// DefaultCssVars returns the default theme token set.
// Colors are inspired by the official palettes of Go, WebAssembly, JavaScript and HTML5
// — the four pillars of the tinywasm ecosystem.
//
//   - Secondary  (#00ADD8) → Go cyan
//   - Selection  (#654FF0) → WebAssembly purple
//   - Hover      (#F7DF1E) → JavaScript yellow
//   - Error      (#E34F26) → HTML5 orange-red
//
// This is a FALLBACK theme. Apps override it by calling CssVars.Render() with their
// own values and injecting the result into <head> before theme.css — CSS cascade
// ensures the app values win.
func DefaultCssVars() CssVars {
    return CssVars{
        Primary:            "#E6EDF3", // light text on dark background
        Secondary:          "#00ADD8", // Go cyan (brand accent)
        Tertiary:           "#8B949E", // muted text / borders
        Quaternary:         "#161B22", // deep background / panels
        Gray:               "#0D1117", // neutral surface (dark)
        Selection:          "#654FF0", // WebAssembly purple
        Hover:              "#F7DF1E", // JavaScript yellow
        Success:            "#3FB950", // Go gopher green
        Error:              "#E34F26", // HTML5 orange-red
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

### 2. `theme.css` (new, embedded in `ssr.theme.go`)

This file is the **default fallback theme** injected once by the site builder into
`<head>`. It is NOT a global app reset — it only defines `:root` CSS custom properties.

Apps that need custom colors call `CssVars.Render()` with their own values and inject
the result **before** this file, or simply after it with higher specificity. CSS cascade
guarantees the app values win.

Default palette is inspired by the official colors of the four pillars of tinywasm:

| Token              | Color   | Origin                  |
|--------------------|---------|-------------------------|
| --color-secondary  | #00ADD8 | Go (cyan)               |
| --color-selection  | #654FF0 | WebAssembly (purple)    |
| --color-hover      | #F7DF1E | JavaScript (yellow)     |
| --color-error      | #E34F26 | HTML5 (orange-red)      |
| --color-success    | #3FB950 | Go gopher green         |
| --color-primary    | #E6EDF3 | Light text (dark base)  |
| --color-gray       | #0D1117 | Dark surface            |
| --color-quaternary | #161B22 | Deeper dark panel       |
| --color-tertiary   | #8B949E | Muted / borders         |

```css
/*
 * tinywasm/dom — canonical design tokens
 *
 * FALLBACK THEME — injected once by the site builder into <head>.
 * Only defines CSS custom properties on :root — no resets, no global styles.
 *
 * Colors are inspired by Go, WebAssembly, JavaScript and HTML5 official palettes.
 *
 * Apps override by injecting their own CssVars.Render() output into <head>.
 * Dark mode works without JS: automatic via prefers-color-scheme,
 * manual override via [data-theme="dark"|"light"] on <html>.
 */

/* ── Default (dark base, Go/WASM inspired) ───────────────────── */
:root {
  --color-primary:    #E6EDF3; /* light text on dark bg            */
  --color-secondary:  #00ADD8; /* Go cyan — brand accent            */
  --color-tertiary:   #8B949E; /* muted text / borders             */
  --color-quaternary: #161B22; /* deep background / panels         */
  --color-gray:       #0D1117; /* neutral dark surface             */
  --color-selection:  #654FF0; /* WebAssembly purple               */
  --color-hover:      #F7DF1E; /* JavaScript yellow                */
  --color-success:    #3FB950; /* Go gopher green                  */
  --color-error:      #E34F26; /* HTML5 orange-red                 */

  --menu-width-collapsed: 64px;
  --menu-width-expanded:  250px;

  --title-height:    8vh;
  --content-height:  89vh;
  --controls-height: 3vh;

  --mag-pri: 0.5rem;
  --mag-sec: 0.2rem;
  --mag-cua: 0.2rem;
}

/* ── Automatic light (OS preference, no JS required) ─────────── */
@media (prefers-color-scheme: light) {
  :root:not([data-theme="dark"]) {
    --color-primary:    #1C1C1E; /* dark text on light bg           */
    --color-secondary:  #00ADD8; /* Go cyan stays                   */
    --color-tertiary:   #6E6E73; /* muted                           */
    --color-quaternary: #F2F2F7; /* light panel                     */
    --color-gray:       #FFFFFF; /* white surface                   */
    --color-selection:  #654FF0; /* WASM purple stays               */
    --color-hover:      #B8860B; /* darker JS yellow (readable)     */
  }
}

/* ── Manual dark override ([data-theme="dark"] on <html>) ────── */
[data-theme="dark"] {
  --color-primary:    #E6EDF3;
  --color-secondary:  #00ADD8;
  --color-tertiary:   #8B949E;
  --color-quaternary: #161B22;
  --color-gray:       #0D1117;
  --color-selection:  #654FF0;
  --color-hover:      #F7DF1E;
}

/* ── Manual light override ([data-theme="light"] on <html>) ───── */
[data-theme="light"] {
  --color-primary:    #1C1C1E;
  --color-secondary:  #00ADD8;
  --color-tertiary:   #6E6E73;
  --color-quaternary: #F2F2F7;
  --color-gray:       #FFFFFF;
  --color-selection:  #654FF0;
  --color-hover:      #B8860B;
}
```

---

### 3. `ssr.theme_test.go` (new, build tag `!wasm`)

Test that:
- `DefaultCssVars().Render()` contains `:root {`
- All 17 tokens are present in the output
- An empty field is omitted from output
- `CssVars` with a custom `Primary` overrides only that token

---

## Dependencies

`ssr.theme.go` uses `github.com/tinywasm/fmt` (already a dependency of `tinywasm/dom`).
No new external dependencies.

## go.mod changes

None required — `tinywasm/fmt` is already in `go.mod`.

## Execution order

1. Create `ssr.theme.go` (includes CssVars + ThemeCSS embed)
2. Create `theme.css`
3. Create `ssr.theme_test.go`
5. Run `go test ./...` — must pass
6. Commit: `feat: add CssVars and theme.css design token standard`

## Notes for Jules

- Do NOT use `reflect` in `ssr.theme.go`. The explicit pairs slice is intentional.
- `ThemeCSS` is a `string`, not a `CSSProvider`. The site builder injects it once.
- The `CssVars.Render()` output is used when a project wants to override tokens
  programmatically (e.g. per-tenant theming). The `theme.css` file is the static default.
- Dark mode uses `:root:not([data-theme="light"])` inside the media query so a user who
  explicitly chose light mode is never overridden by the OS dark preference.
- No JS is required for dark mode to work. The `[data-theme]` attribute is optional.
