package dom

import (
	"testing"
)

func TestSignalString(t *testing.T) {
	s := NewString("hello")
	if s.Get() != "hello" {
		t.Errorf("Expected hello, got %s", s.Get())
	}

	notified := false
	s.subscribe(func() {
		notified = true
	})

	s.Set("world")
	if s.Get() != "world" {
		t.Errorf("Expected world, got %s", s.Get())
	}
	if !notified {
		t.Error("Subscriber not notified")
	}

	notified = false
	s.Set("world") // No-op
	if notified {
		t.Error("Subscriber notified on no-op")
	}

	s.Update(func(v string) string {
		return v + "!"
	})
	if s.Get() != "world!" {
		t.Errorf("Expected world!, got %s", s.Get())
	}
}

func TestSignalBool(t *testing.T) {
	s := NewBool(false)
	if s.Get() != false {
		t.Error("Expected false")
	}

	s.Toggle()
	if s.Get() != true {
		t.Error("Expected true after toggle")
	}

	s.Set(true) // No-op
}

func TestDeriveString(t *testing.T) {
	s1 := NewString("a")
	s2 := NewString("b")
	d := DeriveString(func() string {
		return s1.Get() + s2.Get()
	})

	if d.Get() != "ab" {
		t.Errorf("Expected ab, got %s", d.Get())
	}

	s1.Set("c")
	if d.Get() != "cb" {
		t.Errorf("Expected cb, got %s", d.Get())
	}

	s2.Set("d")
	if d.Get() != "cd" {
		t.Errorf("Expected cd, got %s", d.Get())
	}
}

func TestDeriveDynamicTracking(t *testing.T) {
	cond := NewBool(true)
	s1 := NewString("a")
	s2 := NewString("b")

	count := 0
	d := DeriveString(func() string {
		count++
		if cond.Get() {
			return s1.Get()
		}
		return s2.Get()
	})

	if d.Get() != "a" {
		t.Errorf("Expected a, got %s", d.Get())
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	s1.Set("aa")
	if d.Get() != "aa" {
		t.Errorf("Expected aa, got %s", d.Get())
	}

	s2.Set("bb") // Should NOT trigger d because cond is true
	if d.Get() != "aa" {
		t.Errorf("Expected aa, got %s", d.Get())
	}

	cond.Set(false)
	if d.Get() != "bb" {
		t.Errorf("Expected bb, got %s", d.Get())
	}

	s1.Set("aaa") // Should NOT trigger d anymore
	if d.Get() != "bb" {
		t.Errorf("Expected bb, got %s", d.Get())
	}

	s2.Set("bbb") // Should trigger d
	if d.Get() != "bbb" {
		t.Errorf("Expected bbb, got %s", d.Get())
	}
}

func TestNilSignals(t *testing.T) {
	var s *SignalString
	if s.Get() != "" {
		t.Error("Nil SignalString Get should return empty string")
	}
	s.Set("test") // Should not panic
	s.Update(func(v string) string { return v })

	var b *SignalBool
	if b.Get() != false {
		t.Error("Nil SignalBool Get should return false")
	}
	b.Set(true) // Should not panic
	b.Toggle()

	var n *SignalNodes
	if n.Get() != nil {
		t.Error("Nil SignalNodes Get should return nil")
	}
	n.Set(nil) // Should not panic
}
