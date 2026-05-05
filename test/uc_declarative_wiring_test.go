//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
)

// R1: .On() in Render() captures state updated after Update()
type StateCapturer struct {
	dom.Element
	Value string
	LastCaptured string
}

func (c *StateCapturer) Render() *dom.Element {
	return dom.Div(
		dom.Button().ID("btn-r1").On("click", func(e dom.Event) {
			c.LastCaptured = c.Value
		}),
	)
}

func TestR1StateCapture(t *testing.T) {
	SetupDOM(t)
	c := &StateCapturer{Value: "v1"}
	dom.Render("root", c)

	c.Value = "v2"
	c.Update()

	TriggerEvent("btn-r1", "click", "")
	if c.LastCaptured != "v2" {
		t.Errorf("R1: expected captured value 'v2', got %q", c.LastCaptured)
	}
}

// R3: Race conditions / Re-wiring during Update()
type RewireComp struct {
	dom.Element
	Fired int
}

func (c *RewireComp) Render() *dom.Element {
	return dom.Div(
		dom.Button().ID("btn-r3").On("click", func(e dom.Event) {
			c.Fired++
			c.Update()
		}),
	)
}

func TestR3Rewiring(t *testing.T) {
	SetupDOM(t)
	c := &RewireComp{}
	dom.Render("root", c)

	for i := 1; i <= 5; i++ {
		TriggerEvent("btn-r3", "click", "")
		if c.Fired != i {
			t.Errorf("R3 iteration %d: expected Fired=%d, got %d", i, i, c.Fired)
		}
	}
}
