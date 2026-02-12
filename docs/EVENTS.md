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

    // TargetID returns the ID of the event's target element.
    TargetID() string
}
```

## Usage Examples

### 1. Handling Button Clicks

```go
dom.Button("Click me").
    On("click", func(e dom.Event) {
        // Stop the click from bubbling to parents
        e.StopPropagation()
        
        // Perform action
        println("Button clicked!")
    })
```

### 2. Handling Form Input

Use `TargetValue()` to easily get the new value from an input field.

```go
dom.Text("username").
    On("input", func(e dom.Event) {
        newValue := e.TargetValue()
        println("User typed:", newValue)
    })
```

### 3. Preventing Form Submission

You can use the generic `On("submit", ...)` or the specialized `OnSubmit(...)` for `*FormEl`.

```go
dom.Form(
    dom.Text("query"),
    dom.Button("Search"),
).OnSubmit(func(e dom.Event) {
    e.PreventDefault() // Handled automatically by some wrappers, but safe to call
    handleSearch()
})
```
