package tinydom

// New returns the platform-specific implementation of the DOM interface.
func New(log func(v ...any)) DOM {
	return newDom(log)
}
