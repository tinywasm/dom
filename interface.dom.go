package dom

// DOM is the main entry point for interacting with the browser.
// It is designed to be injected into your components.
type DOM interface {
	// Render injecta un componente en un elemento padre.
	// 1. Llama a componente.Render() (si es ViewRenderer) o componente.RenderHTML()
	// 2. Establece el contenido del elemento padre (buscado por parentID)
	// 3. Llama a componente.OnMount() para enlazar eventos
	Render(parentID string, component Component) error

	// Append injecta un componente DESPUÉS del último hijo del elemento padre.
	// Útil para listas dinámicas.
	Append(parentID string, component Component) error

	// OnHashChange registra un listener para cambios en el hash de la URL.
	OnHashChange(handler func(hash string))

	// GetHash devuelve el hash actual de la URL (ej. "#help").
	GetHash() string

	// SetHash actualiza el hash de la URL.
	SetHash(hash string)

	// Update re-renderiza el componente en su posición actual en el DOM.
	Update(component Component) error

	// Log provides logging functionality using the log function passed to New.
	Log(v ...any)
}

// Component is the minimal interface for components.
// All components must implement this for both SSR (backend) and WASM (frontend).
type Component interface {
	GetID() string
	SetID(id string)
	RenderHTML() string
	Children() []Component
}

// ViewRenderer returns a Node tree for declarative UI.
type ViewRenderer interface {
	Render() *Element
}

// Mountable is an optional interface for components that need initialization logic.
type Mountable interface {
	OnMount()
}

// Updatable is an optional interface for components that need update logic.
type Updatable interface {
	OnUpdate()
}

// Unmountable is an optional interface for components that need cleanup logic.
type Unmountable interface {
	OnUnmount()
}

// EventHandler represents a DOM event handler in the declarative builder.
type EventHandler struct {
	Name    string
	Handler func(Event)
}
