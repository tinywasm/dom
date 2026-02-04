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

// Element represents a DOM node. It provides methods for direct manipulation and event binding.
//
// All content methods (SetText, SetHTML, AppendHTML, SetAttr, SetValue) accept variadic arguments
// and support multiple input types:
//   - Strings: Concatenated without spaces
//   - Numbers: Converted to strings
//   - Format strings: Printf-style formatting with % specifiers
//   - Localized strings: Using D.* dictionary for multilingual support
//
// For more information about translation and multilingual support, see:
// https://github.com/tinywasm/fmt/blob/main/docs/TRANSLATE.md
//
// Examples:
//
//	elem.SetText("Hello ", "World")           // -> "Hello World"
//	elem.SetHTML("<div>", "content", "</div>") // -> "<div>content</div>"
//	elem.SetAttr("class", "btn-", 42)         // -> "btn-42"
//	elem.SetText(D.Hello)                     // -> "Hello" (EN) or "Hola" (ES)
//	elem.SetHTML("<h1>%v</h1>", 42)           // -> "<h1>42</h1>"
type Element interface {
	// --- Content ---

	// SetText sets the text content of the element.
	// Accepts variadic arguments that are concatenated without spaces.
	//
	// Examples:
	//   elem.SetText("Count: ", 42)              // -> "Count: 42"
	//   elem.SetText(D.Hello, " ", D.User)       // -> "Hello User" (localized)
	SetText(v ...any)

	// SetHTML sets the inner HTML of the element.
	// Accepts variadic arguments that are concatenated without spaces.
	// Supports format strings with % specifiers.
	//
	// Examples:
	//   elem.SetHTML("<div>", "content", "</div>")  // -> "<div>content</div>"
	//   elem.SetHTML("<h1>%v</h1>", 42)             // -> "<h1>42</h1>"
	//   elem.SetHTML("<span>%L</span>", D.Hello)    // -> "<span>Hello</span>" (localized)
	SetHTML(v ...any)

	// AppendHTML adds HTML to the end of the element's content.
	// Useful for adding items to a list without re-rendering the whole list.
	// Accepts variadic arguments that are concatenated without spaces.
	//
	// Examples:
	//   elem.AppendHTML("<li>", item, "</li>")
	//   elem.AppendHTML("<div class='%s'>%v</div>", "item", count)
	AppendHTML(v ...any)

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
	// Accepts variadic arguments that are concatenated without spaces.
	//
	// Examples:
	//   elem.SetAttr("id", "item-", 42)           // -> id="item-42"
	//   elem.SetAttr("href", "/page/", pageNum)   // -> href="/page/5"
	//   elem.SetAttr("title", D.Hello)            // -> title="Hello" (localized)
	SetAttr(key string, v ...any)

	// GetAttr retrieves an attribute value.
	GetAttr(key string) string

	// RemoveAttr removes an attribute.
	RemoveAttr(key string)

	// --- Forms ---

	// Value returns the current value of an input/textarea/select.
	Value() string

	// SetValue sets the value of an input/textarea/select.
	// Accepts variadic arguments that are concatenated without spaces.
	//
	// Examples:
	//   elem.SetValue("default value")
	//   elem.SetValue("Item ", 42)                // -> "Item 42"
	SetValue(v ...any)

	// --- Checkboxes ---

	// Checked returns the current checked state of a checkbox or radio button.
	Checked() bool

	// SetChecked sets the checked state of a checkbox or radio button.
	SetChecked(checked bool)

	// --- Events ---

	// Click registers a click event handler.
	// The handler is automatically tracked and removed when the component is unmounted.
	Click(handler func(event Event))

	// On registers a generic event handler (e.g., "change", "input", "keydown").
	On(eventType string, handler func(event Event))

	// Focus sets focus to the element.
	Focus()
}
