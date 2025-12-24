# Event Handling

TinyDOM provides a simple `Event` interface that wraps the underlying browser event. This allows you to handle user interactions without dealing with `syscall/js` values directly.

## The Event Interface

```go
type Event interface {
    // PreventDefault prevents the default action (e.g., form submission).
    PreventDefault()

    // StopPropagation stops the event from bubbling up the DOM tree.
    StopPropagation()

    // TargetValue returns the value of the event's target element.
    // Extremely useful for <input>, <textarea>, and <select> changes.
    TargetValue() string
}
```

## Usage Examples

### 1. Handling Button Clicks

```go
btn.Click(func(e dom.Event) {
    // Stop the click from bubbling to parents
    e.StopPropagation()
    
    // Perform action
    println("Button clicked!")
})
```

### 2. Handling Form Input

Use `TargetValue()` to easily get the new value from an input field.

```go
inputEl.On("input", func(e dom.Event) {
    newValue := e.TargetValue()
    println("User typed:", newValue)
})
```

### 3. Preventing Form Submission

```go
formEl.On("submit", func(e dom.Event) {
    // Prevent the browser from reloading the page
    e.PreventDefault()
    
    // Handle submission via AJAX/Fetch
    submitData()
})
```
