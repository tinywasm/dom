//go:build !wasm

package dom

import (
	"strings"
	"testing"
)

func TestCssVars_Render(t *testing.T) {
	t.Run("DefaultCssVars renders all tokens", func(t *testing.T) {
		vars := DefaultCssVars()
		got := vars.RenderCSS()

		expectations := []string{
			":root {",
			"--color-primary: #1C1C1E;",
			"--color-secondary: #00ADD8;",
			"--color-tertiary: #6E6E73;",
			"--color-quaternary: #F2F2F7;",
			"--color-gray: #FFFFFF;",
			"--color-selection: #654FF0;",
			"--color-hover: #F7DF1E",
			"--color-success: #3FB950;",
			"--color-error: #E34F26;",
			"--menu-width-collapsed: 64px;",
			"--menu-width-expanded: 250px;",
			"--title-height: 8vh;",
			"--content-height: 89vh;",
			"--controls-height: 3vh;",
			"--mag-pri: 0.5rem;",
			"--mag-sec: 0.2rem;",
			"--mag-cua: 0.2rem;",
			"}",
		}

		for _, exp := range expectations {
			if !strings.Contains(got, exp) {
				t.Errorf("expected output to contain %q, but it didn't\nGot:\n%s", exp, got)
			}
		}
	})

	t.Run("Omit empty fields", func(t *testing.T) {
		vars := CssVars{
			Primary: "#000",
		}
		got := vars.RenderCSS()

		if !strings.Contains(got, "--color-primary: #000;") {
			t.Errorf("expected output to contain primary color, got %q", got)
		}

		if strings.Contains(got, "--color-secondary:") {
			t.Errorf("expected output NOT to contain secondary color when empty, got %q", got)
		}
	})

	t.Run("Custom values override defaults", func(t *testing.T) {
		vars := DefaultCssVars()
		vars.Primary = "#FF0000"
		got := vars.RenderCSS()

		if !strings.Contains(got, "--color-primary: #FF0000;") {
			t.Errorf("expected output to contain custom primary color #FF0000, got %q", got)
		}

		// Ensure another default is still there
		if !strings.Contains(got, "--color-secondary: #00ADD8;") {
			t.Errorf("expected output to still contain default secondary color, got %q", got)
		}
	})
}

func TestThemeCSS_Embedded(t *testing.T) {
	if ThemeCSS == "" {
		t.Fatal("ThemeCSS should not be empty")
	}
	if !strings.Contains(ThemeCSS, ":root") {
		t.Error("ThemeCSS should contain :root block")
	}
	if !strings.Contains(ThemeCSS, "@media (prefers-color-scheme: dark)") {
		t.Error("ThemeCSS should contain dark mode media query")
	}
	if !strings.Contains(ThemeCSS, "--mag-cua") {
		t.Error("ThemeCSS should contain --mag-cua token")
	}
	if strings.Contains(ThemeCSS, ":root:not([data-theme=\"light\"])") {
		t.Error("ThemeCSS should not use high-specificity selector in media query")
	}
}
