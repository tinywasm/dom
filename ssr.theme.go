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
// Use RenderCSS() to generate the :root { } block for SSR injection.
type CssVars struct {
	Primary            string `css:"--color-primary"`
	Secondary          string `css:"--color-secondary"`
	Tertiary           string `css:"--color-tertiary"`
	Quaternary         string `css:"--color-quaternary"`
	Gray               string `css:"--color-gray"`
	Selection          string `css:"--color-selection"`
	Hover              string `css:"--color-hover"`
	Success            string `css:"--color-success"`
	Error              string `css:"--color-error"`
	MenuWidthCollapsed string `css:"--menu-width-collapsed"`
	MenuWidthExpanded  string `css:"--menu-width-expanded"`
	TitleHeight        string `css:"--title-height"`
	ContentHeight      string `css:"--content-height"`
	ControlsHeight     string `css:"--controls-height"`
	SpacingPrimary     string `css:"--mag-pri"`
	SpacingSecondary   string `css:"--mag-sec"`
	SpacingQuaternary  string `css:"--mag-cua"`
}

// DefaultCssVars returns the default theme token set.
// Colors are inspired by the official palettes of Go, WebAssembly, JavaScript and HTML5.
//
// This is a FALLBACK theme. Apps override it by calling CssVars.Render() with their
// own values and injecting the result into <head> before theme.css — CSS cascade
// ensures the app values win.
func DefaultCssVars() CssVars {
	return CssVars{
		Primary:            "#1C1C1E", // dark text on light bg
		Secondary:          "#00ADD8", // Go cyan (brand accent)
		Tertiary:           "#6E6E73", // muted
		Quaternary:         "#F2F2F7", // light panel
		Gray:               "#FFFFFF", // white surface
		Selection:          "#654FF0", // WebAssembly purple
		Hover:              "#F7DF1E", // darker JS yellow (readable)
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

// RenderCSS returns the CSS :root { } block with all non-empty tokens.
// Safe to call from SSR. Does NOT use reflect — iterates fields explicitly.
func (c CssVars) renderCSS() string {
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
