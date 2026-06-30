package dom

// sub is a single subscription: a stable id (for removal without reflect) and
// the callback to run on change.
type sub struct {
	id uint64
	fn func()
}

// notify invokes each subscriber over a snapshot of the slice. Subscribers
// (e.g. a DeriveString updater) unsubscribe and re-subscribe themselves while
// running, which mutates the live subs slice. Ranging the live slice would skip
// the element that shifts into the freed index — leaving a sibling binding one
// update behind. Iterating a copy makes every subscriber present at change time
// fire exactly once.
func notify(subs []sub) {
	snapshot := make([]sub, len(subs))
	copy(snapshot, subs)
	for _, s := range snapshot {
		if s.fn != nil {
			s.fn()
		}
	}
}

// removeSub drops the subscription with the given id from subs, returning the
// new slice. No-op if the id is absent (already removed).
func removeSub(subs []sub, id uint64) []sub {
	for i, s := range subs {
		if s.id == id {
			return append(subs[:i], subs[i+1:]...)
		}
	}
	return subs
}

// SignalString is an observable string cell. UI text/attr/input state lives here. Explicit Get/Set.
type SignalString struct {
	v      string
	subs   []sub // binding callbacks; invoked on change
	nextID uint64
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
	notify(s.subs)
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
	s.nextID++
	id := s.nextID
	s.subs = append(s.subs, sub{id: id, fn: fn})
	return func() { s.subs = removeSub(s.subs, id) }
}

// SignalBool — same shape for class/attr toggles and Show conditions.
type SignalBool struct {
	v      bool
	subs   []sub
	nextID uint64
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
	notify(s.subs)
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
	s.nextID++
	id := s.nextID
	s.subs = append(s.subs, sub{id: id, fn: fn})
	return func() { s.subs = removeSub(s.subs, id) }
}

// SignalNodes is an observable list of rendered rows. No generics; the component builds the Elements.
type SignalNodes struct {
	v      []*Element
	subs   []sub
	nextID uint64
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
	notify(s.subs)
}

func (s *SignalNodes) subscribe(fn func()) (unsub func()) {
	if s == nil {
		return func() {}
	}
	s.nextID++
	id := s.nextID
	s.subs = append(s.subs, sub{id: id, fn: fn})
	return func() { s.subs = removeSub(s.subs, id) }
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
