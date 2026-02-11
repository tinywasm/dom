package dom

// Event represents a DOM event.
type Event interface {
	// PreventDefault prevents the default action of the event.
	PreventDefault()
	// StopPropagation stops the event from bubbling up the DOM tree.
	StopPropagation()
	// TargetValue returns the value of the event's target element.
	// Useful for input, textarea, and select elements.
	TargetValue() string
	// TargetID returns the ID of the event's target element.
	TargetID() string
}
