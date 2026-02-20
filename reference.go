package dom

// Reference represents a reference to a DOM node. It provides methods for reading and interaction.
type Reference interface {
	// --- Attributes ---

	// GetAttr retrieves an attribute value.
	GetAttr(key string) string

	// --- Forms ---

	// Value returns the current value of an input/textarea/select.
	Value() string

	// --- Checkboxes ---

	// Checked returns the current checked state of a checkbox or radio button.
	Checked() bool

	// --- Events ---

	// On registers a generic event handler (e.g., "click", "change", "input", "keydown").
	On(eventType string, handler func(event Event))

	// Focus sets focus to the element.
	Focus()
}
