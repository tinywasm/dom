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
