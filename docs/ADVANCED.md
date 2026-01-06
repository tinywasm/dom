# Advanced Component Patterns

## Dynamic Lists (Node Manipulation)

When working with lists, you often want to add or remove items without re-rendering the entire list. This preserves the state (focus, scroll position) of other items.

```go
type TodoList struct {
    dom   tinywasm/dom.DOM
    id    string
    items []*TodoItem
}

func (l *TodoList) RenderHTML() string {
    // Initial render might be empty or have initial items
    return `<ul id="` + l.id + `"></ul>`
}

func (l *TodoList) AddItem(label string) {
    // 1. Create new component
    newItem := NewTodoItem(l.dom, l.id + "-item-" + uniqueID(), label)
    l.items = append(l.items, newItem)

    // 2. Append HTML to the list container
    // This is more efficient than re-rendering the whole <ul>
    listEl := l.dom.Get(l.id)
    listEl.AppendHTML(newItem.RenderHTML())

    // 3. Mount the new item (bind events)
    newItem.OnMount()
}

func (l *TodoList) RemoveItem(itemID string) {
    // 1. Unmount handles everything:
    // - Finds the element by ID
    // - Removes it from the browser DOM
    // - Cleans up all event listeners associated with this ID
    l.dom.Unmount(itemID)

    // 2. Update internal state (remove from slice)...
}
```

## Decoupled Components (Interface Segregation)

You don't always need to import `tinywasm/dom` or depend on the full `DOM` interface. You can define narrow interfaces for what your component actually needs.

```go
// This component only needs to update text. It doesn't care about Mounting or Unmounting.
type TextSetter interface {
    Get(id string) tinywasm/dom.Element
}

type StatusLabel struct {
    dom TextSetter // Narrow dependency
    id  string
}

func (s *StatusLabel) Update(msg string) {
    s.dom.Get(s.id).SetText(msg)
}
```

This makes `StatusLabel` easier to test (you only need to mock `Get`) and less coupled to the framework core.
