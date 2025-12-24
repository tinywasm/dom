//go:build wasm

package dom

// Mountable extends Component with lifecycle hooks for WASM.
// Components that need interactivity should implement this interface.
type Mountable interface {
	Component

	// OnMount is called after the HTML has been injected into the DOM.
	// The component can now bind events and interact with elements using the global API.
	OnMount()

	// OnUnmount is called before the component is removed from the DOM.
	OnUnmount()
}
