package dom

// BaseComponent is a helper struct that implements the Identifiable interface.
// Users can embed this in their components to automatically handle ID management.
type BaseComponent struct {
	id string
}

// ID returns the component's unique identifier.
func (c *BaseComponent) ID() string {
	return c.id
}

// SetID sets the component's unique identifier.
func (c *BaseComponent) SetID(id string) {
	c.id = id
}

// Children returns the component's children (nil by default).
func (c *BaseComponent) Children() []Component {
	return nil
}

// RenderHTML returns an empty string by default, satisfying the HTMLRenderer interface.
func (c *BaseComponent) RenderHTML() string {
	return ""
}
