package dom

// Reference represents a reference to a DOM node. It provides methods for reading and interaction.
type Reference interface {
	// --- Attributes ---

	// GetAttr retrieves an attribute value.
	GetAttr(key string) string

	// --- Forms ---

	// Value returns the current value of an input/textarea/select.
	Value() string

	// SetValue sets element.value (inputs, textarea, select).
	SetValue(value string)

	// SetAttr calls element.setAttribute(key, value).
	// Use empty string for boolean attributes (e.g., SetAttr("disabled", "")).
	SetAttr(key, value string)

	// RemoveAttr calls element.removeAttribute(key).
	RemoveAttr(key string)

	// SetText sets element.textContent.
	// Safe for plain text — does not parse HTML.
	SetText(text string)

	// --- Checkboxes ---

	// Checked returns the current checked state of a checkbox or radio button.
	Checked() bool

	// --- Events ---

	// On registers a generic event handler (e.g., "click", "change", "input", "keydown").
	On(eventType string, handler func(event Event))

	// Focus sets focus to the element.
	Focus()

	// ScrollIntoView smooth-scrolls the element into view (e.g. to jump a
	// horizontal scroll-snap container to a different panel programmatically —
	// the browser resolves the final resting position against any
	// scroll-snap-align on this element and its container).
	ScrollIntoView()
}
