//go:build wasm

package dom_test

import (
	"testing"

	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

type SearchOption struct {
	ID          string
	Label       string
	Description string
}

type SelectSearch struct {
	dom.Element // value embed (TinyGo heap constraint)

	Placeholder string
	Options     []SearchOption
	OnSelect    func(id, description string)
	OnSearch    func(term string) []SearchOption

	selectedLabel string
	filterTerm    string
	isOpen        bool
}

func (c *SelectSearch) Render() *dom.Element {
	headerText := c.Placeholder
	if c.selectedLabel != "" {
		headerText = c.selectedLabel
	}
	if headerText == "" {
		headerText = "Select..."
	}

	toggle := dom.Input("checkbox").Class("ss-toggle").ID("ss-toggle-id")
	if c.isOpen {
		toggle.Attr("checked", "")
	}

	header := dom.Label().
		For(toggle). // typed pairing — sin strings
		Class("ss-header").
		Text(headerText).
		Add(dom.Svg(dom.Use().Attr("href", "#ss-arrow-down")).Class("ss-icon"))

	search := dom.Input("search").
		ID("ss-search-id").
		Class("ss-search").
		Attr("placeholder", "Search...").
		Attr("value", c.filterTerm).
		On("input", c.onSearchInput) // handler junto al elemento

	list := dom.Div().Class("ss-options")
	filterTerm := fmt.Convert(c.filterTerm).ToLower().String()
	for _, opt := range c.Options {
		if !c.matches(opt, filterTerm) {
			continue
		}
		opt := opt
		item := dom.Div().Class("ss-option").
			ID("ss-opt-"+opt.ID).
			Attr("data-id", opt.ID).
			On("click", func(e dom.Event) { c.selectOption(opt) }).
			Add(dom.Span().Class("ss-label").Text(opt.Label))
		if opt.Description != "" {
			item.Add(dom.Span().Class("ss-desc").Text(opt.Description))
		}
		list.Add(item)
	}

	return dom.Div().Class("ss-box").
		Add(toggle).
		Add(header).
		Add(dom.Div().Class("ss-dropdown").
			Add(search).
			Add(list))
}

func (c *SelectSearch) onSearchInput(e dom.Event) {
	c.filterTerm = e.TargetValue()
	if len(c.filteredOptions()) == 0 && c.OnSearch != nil {
		c.Options = c.OnSearch(c.filterTerm)
	}
	c.Update()
}

func (c *SelectSearch) selectOption(opt SearchOption) {
	c.selectedLabel = opt.Label
	c.isOpen = false
	if c.OnSelect != nil {
		c.OnSelect(opt.ID, opt.Description)
	}
	c.Update()
}

func (c *SelectSearch) matches(opt SearchOption, term string) bool {
	if term == "" {
		return true
	}
	return fmt.Contains(fmt.Convert(opt.Label).ToLower().String(), term) ||
		fmt.Contains(fmt.Convert(opt.Description).ToLower().String(), term)
}

func (c *SelectSearch) filteredOptions() []SearchOption {
	term := fmt.Convert(c.filterTerm).ToLower().String()
	out := make([]SearchOption, 0, len(c.Options))
	for _, o := range c.Options {
		if c.matches(o, term) {
			out = append(out, o)
		}
	}
	return out
}

func TestSelectSearch(t *testing.T) {
	SetupDOM(t)

	selectedID := ""
	c := &SelectSearch{
		Placeholder: "Choose one",
		Options: []SearchOption{
			{ID: "opt1", Label: "Option 1", Description: "First option"},
			{ID: "opt2", Label: "Option 2", Description: "Second option"},
		},
		OnSelect: func(id, desc string) {
			selectedID = id
		},
	}

	if err := dom.Render("root", c); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify label points to toggle
	// We added IDs for testing purposes in this test component
	_, ok := GetRef("ss-toggle-id")
	if !ok {
		t.Fatal("toggle not found")
	}

	// Test Search
	TriggerEvent("ss-search-id", "input", "Option 2")
	if c.filterTerm != "Option 2" {
		t.Errorf("expected filterTerm='Option 2', got %q", c.filterTerm)
	}

	// Verify opt1 is filtered out
	_, ok = GetRef("ss-opt-opt1")
	if ok {
		t.Error("opt1 should be filtered out")
	}

	// Test Select
	TriggerEvent("ss-opt-opt2", "click", "")
	if selectedID != "opt2" {
		t.Errorf("expected selectedID='opt2', got %q", selectedID)
	}
	if c.selectedLabel != "Option 2" {
		t.Errorf("expected selectedLabel='Option 2', got %q", c.selectedLabel)
	}
}
