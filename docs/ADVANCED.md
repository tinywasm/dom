# Advanced Component Patterns

## Dynamic Lists (Node Manipulation)

When working with lists, you often want to add or remove items without re-rendering the entire list. This preserves the state (focus, scroll position) of other items.

```go
import (
	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

type TodoList struct {
	*dom.Element
	items []string
}

func (l *TodoList) Render() *dom.Element {
	var children []any
	for _, item := range l.items {
		children = append(children, dom.Li(item))
	}
	return dom.Ul(children...)
}

func (l *TodoList) AddItem(label string) {
	l.items = append(l.items, label)
	l.Update()
}
```

### Partial Updates (Append)

If the list is very large, you can use `dom.Append` to add a single item without re-rendering the entire list. This preserves scroll position and focus of existing elements.

```go
func (l *TodoList) AddItemEfficiently(label string) {
	l.items = append(l.items, label)
	
	// Append only the new item to the DOM parent
	itemComp := dom.Li(label)
	dom.Append(l.GetID(), itemComp)
}
```

## Decoupled Components

Instead of relying on imperative DOM selection, define components that receive data via their `Render` cycle. This makes them easier to test and more predictable.

```go
type StatusLabel struct {
    *dom.Element
    status string
}

func (s *StatusLabel) SetStatus(status string) {
    s.status = status
    s.Update() // Declarative update
}

func (s *StatusLabel) Render() *dom.Element {
    return dom.Span(s.status).Class("status-label")
}
```
