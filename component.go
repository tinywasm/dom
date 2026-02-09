package dom

// BaseComponent is a helper struct that implements the Identifiable interface.
// Users can embed this in their components to automatically handle ID management.
type BaseComponent struct {
	id     string
	prefix string // Optional semantic prefix for debugging
}

// GetID returns the component's unique identifier.
func (c *BaseComponent) GetID() string {
	if c.id == "" {
		c.id = c.generateID()
	}
	return c.id
}

// SetID sets the component's unique identifier.
func (c *BaseComponent) SetID(id string) {
	c.id = id
}

func (c *BaseComponent) generateID() string {
	if c.prefix != "" {
		return c.prefix + "-" + generateID()
	}
	return generateID()
}

// Chainable lifecycle helpers

// Mount injects the component into a parent element.
func (c *BaseComponent) Mount(parentID string) *BaseComponent {
	Render(parentID, c)
	return c
}

// Render re-renders the component in place.
// This is used for chaining, e.g., component.SetState(...).Render()
func (c *BaseComponent) Render() *BaseComponent {
	Update(c)
	return c
}

// Update triggers a re-render of the component.
func (c *BaseComponent) Update() error {
	return Update(c)
}

// Unmount removes the component from the DOM.
func (c *BaseComponent) Unmount() {
	Unmount(c)
}

// Children returns the component's children (nil by default).
func (c *BaseComponent) Children() []Component {
	return nil
}

// RenderHTML returns an empty string by default, satisfying the HTMLRenderer interface.
func (c *BaseComponent) RenderHTML() string {
	return ""
}
