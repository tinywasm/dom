package dom

// tinyDOM contains shared functionality between backend and WASM implementations.
type tinyDOM struct {
	log func(v ...any)
}

// New returns the platform-specific implementation of the DOM interface.
func New(log func(v ...any)) DOM {
	td := &tinyDOM{log: log}
	return newDom(td)
}

// Log provides logging functionality using the log function passed to New.
func (t *tinyDOM) Log(v ...any) {
	if t.log != nil {
		t.log(v...)
	}
}
