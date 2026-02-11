# Advanced Component Patterns

## Dynamic Lists (Node Manipulation)

When working with lists, you often want to add or remove items without re-rendering the entire list. This preserves the state (focus, scroll position) of other items.

```go
import (
	"github.com/tinywasm/dom"
)

type TodoList struct {
	*dom.Element
	items []*TodoItem
}

func (l *TodoList) Render() dom.Node {
	// Initial render might be empty or have initial items.
	// We use a container that we can append to later.
	// dom.Ul root automatically gets l.GetID() in Render cycle.
	return dom.Ul().ToNode()
}

func (l *TodoList) AddItem(label string) {
	// 1. Create new component
	newItem := NewTodoItem(l.GetID() + "-item-" + uniqueID(), label)
	l.items = append(l.items, newItem)

	// 2. Append the new item to the list
	// dom.Append renders the component and injects it at the end of the parent
	// while preserving the existing DOM (and focus).
	// It also automatically handles lifecycle (OnMount / Events).
	dom.Append(l.GetID(), newItem)
}

func (l *TodoList) RemoveItem(item *TodoItem) {
	// 1. Unmount handles everything recursively:
	// - Finds the element by ID
	// - Removes it from the browser DOM
	// - Cleans up all event listeners recursively
	dom.Unmount(item)

	// 2. Update internal state (remove from slice)...
}
```

## Decoupled Components (Interface Segregation)

You don't always need to import `tinywasm/dom` or depend on the full `DOM` interface. You can define narrow interfaces for what your component actually needs.

```go
// This component only needs to find an element.
type ElementFinder interface {
    Get(id string) (dom.Reference, bool)
}

type StatusLabel struct {
    *dom.Element
    dom ElementFinder // Narrow dependency
}

func (s *StatusLabel) UpdateStatus(msg string) {
    // Note: State updates should ideally happen via Render() + dom.Update(s)
    // but direct DOM reading/focus can use the Finder.
}
```

This makes `StatusLabel` easier to test (you only need to mock `Get`) and less coupled to the framework core.
