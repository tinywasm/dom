package dom

// Theme represents the visual theme of the application.
type Theme string

const (
	ThemeAuto  Theme = "auto"  // No override, follows OS preference
	ThemeDark  Theme = "dark"  // Dark mode override
	ThemeLight Theme = "light" // Light mode override
)
