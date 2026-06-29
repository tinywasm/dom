package dom

import "reflect"

// SignalString is an observable string cell. UI text/attr/input state lives here. Explicit Get/Set.
type SignalString struct {
	v    string
	subs []func() // binding callbacks; invoked on change
}

func NewString(v string) *SignalString { return &SignalString{v: v} }

func (s *SignalString) Get() string {
	if currentTracker != nil {
		currentTracker.add(s)
	}
	if s == nil {
		return ""
	}
	return s.v
}

func (s *SignalString) Set(v string) {
	if s == nil || v == s.v {
		return
	}
	s.v = v
	for _, fn := range s.subs {
		if fn != nil {
			fn()
		}
	}
}

func (s *SignalString) Update(fn func(string) string) {
	if s == nil {
		return
	}
	s.Set(fn(s.v))
}

func (s *SignalString) subscribe(fn func()) (unsub func()) {
	if s == nil {
		return func() {}
	}
	s.subs = append(s.subs, fn)
	return func() {
		for i, sub := range s.subs {
			if reflect.ValueOf(sub).Pointer() == reflect.ValueOf(fn).Pointer() {
				s.subs = append(s.subs[:i], s.subs[i+1:]...)
				break
			}
		}
	}
}

// SignalBool — same shape for class/attr toggles and Show conditions.
type SignalBool struct {
	v    bool
	subs []func()
}

func NewBool(v bool) *SignalBool { return &SignalBool{v: v} }

func (s *SignalBool) Get() bool {
	if currentTracker != nil {
		currentTracker.add(s)
	}
	if s == nil {
		return false
	}
	return s.v
}

func (s *SignalBool) Set(v bool) {
	if s == nil || v == s.v {
		return
	}
	s.v = v
	for _, fn := range s.subs {
		if fn != nil {
			fn()
		}
	}
}

func (s *SignalBool) Toggle() {
	if s == nil {
		return
	}
	s.Set(!s.v)
}

func (s *SignalBool) subscribe(fn func()) (unsub func()) {
	if s == nil {
		return func() {}
	}
	s.subs = append(s.subs, fn)
	return func() {
		for i, sub := range s.subs {
			if reflect.ValueOf(sub).Pointer() == reflect.ValueOf(fn).Pointer() {
				s.subs = append(s.subs[:i], s.subs[i+1:]...)
				break
			}
		}
	}
}

// SignalNodes is an observable list of rendered rows. No generics; the component builds the Elements.
type SignalNodes struct {
	v    []*Element
	subs []func()
}

func NewNodes(v ...*Element) *SignalNodes { return &SignalNodes{v: v} }

func (s *SignalNodes) Get() []*Element {
	if currentTracker != nil {
		currentTracker.add(s)
	}
	if s == nil {
		return nil
	}
	return s.v
}

func (s *SignalNodes) Set(v []*Element) {
	if s == nil {
		return
	}
	s.v = v
	for _, fn := range s.subs {
		if fn != nil {
			fn()
		}
	}
}

func (s *SignalNodes) subscribe(fn func()) (unsub func()) {
	if s == nil {
		return func() {}
	}
	s.subs = append(s.subs, fn)
	return func() {
		for i, sub := range s.subs {
			if reflect.ValueOf(sub).Pointer() == reflect.ValueOf(fn).Pointer() {
				s.subs = append(s.subs[:i], s.subs[i+1:]...)
				break
			}
		}
	}
}

// subscribable (UNEXPORTED) — its method is unexported, so only dom's own signals satisfy it.
type subscribable interface {
	subscribe(fn func()) (unsub func())
}

type tracker struct {
	signals []subscribable
}

func (t *tracker) add(s subscribable) {
	for _, sig := range t.signals {
		if sig == s {
			return
		}
	}
	t.signals = append(t.signals, s)
}

var currentTracker *tracker

// DeriveString / DeriveBool: read-only computed cells. Re-run automatically when any signal the
// closure READS changes — no deps argument.
func DeriveString(compute func() string) *SignalString {
	s := NewString("")
	var unsubs []func()
	var updater func()
	updater = func() {
		for _, unsub := range unsubs {
			unsub()
		}
		unsubs = nil

		t := &tracker{}
		prev := currentTracker
		currentTracker = t
		val := compute()
		currentTracker = prev

		for _, sig := range t.signals {
			unsubs = append(unsubs, sig.subscribe(updater))
		}
		s.Set(val)
	}

	updater()
	return s
}

func DeriveBool(compute func() bool) *SignalBool {
	s := NewBool(false)
	var unsubs []func()
	var updater func()
	updater = func() {
		for _, unsub := range unsubs {
			unsub()
		}
		unsubs = nil

		t := &tracker{}
		prev := currentTracker
		currentTracker = t
		val := compute()
		currentTracker = prev

		for _, sig := range t.signals {
			unsubs = append(unsubs, sig.subscribe(updater))
		}
		s.Set(val)
	}

	updater()
	return s
}
