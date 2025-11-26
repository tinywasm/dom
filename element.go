package tinydom

// Event represents a DOM event.
type Event interface {
	// PreventDefault prevents the default action of the event.
	PreventDefault()
	// StopPropagation stops the event from bubbling up the DOM tree.
	StopPropagation()
	// TargetValue returns the value of the event's target element.
	// Useful for input, textarea, and select elements.
	TargetValue() string
}

// Element represents a DOM node. It provides methods for direct manipulation and event binding.
type Element interface {
	// --- Content ---

	// SetText sets the text content of the element.
	SetText(text string)

	// SetHTML sets the inner HTML of the element.
	SetHTML(html string)

	// AppendHTML adds HTML to the end of the element's content.
	// Useful for adding items to a list without re-rendering the whole list.
	AppendHTML(html string)

	// Remove removes the element from the DOM.
	Remove()

	// --- Attributes & Classes ---

	// AddClass adds a CSS class to the element.
	AddClass(class string)

	// RemoveClass removes a CSS class from the element.
	RemoveClass(class string)

	// ToggleClass toggles a CSS class.
	ToggleClass(class string)

	// SetAttr sets an attribute value.
	SetAttr(key, value string)

	// GetAttr retrieves an attribute value.
	GetAttr(key string) string

	// RemoveAttr removes an attribute.
	RemoveAttr(key string)

	// --- Forms ---

	// Value returns the current value of an input/textarea/select.
	Value() string

	// SetValue sets the value of an input/textarea/select.
	SetValue(value string)

	// --- Events ---

	// Click registers a click event handler.
	// The handler is automatically tracked and removed when the component is unmounted.
	Click(handler func(event Event))

	// On registers a generic event handler (e.g., "change", "input", "keydown").
	On(eventType string, handler func(event Event))

	// Focus sets focus to the element.
	Focus()
}
