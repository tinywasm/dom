//go:build wasm

package main

import (
	. "github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

// --- App State & Components ---

type App struct {
	*Element
	currentRoute string
	counter      int
}

func (a *App) Init() {
	// 1. Inject minimal CSS
	css := `
		body { font-family: sans-serif; margin: 0; background: #f4f4f9; color: #333; }
		nav { background: #333; padding: 1rem; display: flex; gap: 1rem; }
		nav a { color: white; text-decoration: none; cursor: pointer; padding: 0.2rem 0.5rem; border-radius: 4px; }
		nav a:hover { background: #555; }
		nav a.active { background: #007bff; }
		.container { padding: 2rem; max-width: 800px; margin: 0 auto; }
		.card { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
		.btn-group { display: flex; gap: 0.5rem; align-items: center; margin-top: 1rem; }
		button { padding: 0.5rem 1rem; cursor: pointer; border: none; border-radius: 4px; background: #007bff; color: white; }
		button:hover { background: #0056b3; }
		.count { font-size: 1.5rem; font-weight: bold; min-width: 3rem; text-align: center; }
	`
	renderStyle(css)

	// 2. Setup Routing
	OnHashChange(func(hash string) {
		a.currentRoute = hash
		a.Update()
	})

	// Initial route
	a.currentRoute = GetHash()
	if a.currentRoute == "" {
		a.currentRoute = "#home"
		SetHash("#home")
	}
}

func (a *App) Render() *Element {
	return Div(
		// Navigation Bar
		Nav(
			NavLink("Home", "#home", a.currentRoute == "#home"),
			NavLink("About", "#about", a.currentRoute == "#about"),
		),

		// Content Area
		Div(
			a.renderRoute(),
		).Class("container"),
	)
}

func (a *App) renderRoute() *Element {
	switch a.currentRoute {
	case "#about":
		return Div(
			H1("About This Library"),
			P("tinywasm/dom is a minimalist, WASM-optimized DOM toolkit for Go."),
			P("It features a JSX-like Builder API, Elm-inspired state management, and no Virtual DOM overhead."),
		).Class("card")
	default: // "#home"
		return Div(
			H1("Counter Example"),
			P("This demonstrates local state updates and hash routing."),
			Div(
				Button("-").On("click", func(e Event) { a.counter--; a.Update() }),
				Span(fmt.Sprint(a.counter)).Class("count"),
				Button("+").On("click", func(e Event) { a.counter++; a.Update() }),
			).Class("btn-group"),
		).Class("card")
	}
}

// --- Helpers ---

func NavLink(text, hash string, active bool) *Element {
	link := A(hash, text).On("click", func(e Event) {
		e.PreventDefault()
		SetHash(hash)
	})
	if active {
		link.Class("active")
	}
	return link
}

func renderStyle(css string) {
	// Inject style into head
	Append("head", Style(css))
}

func main() {
	app := &App{Element: &Element{}, counter: 0}
	app.Init()

	Render("body", app)

	fmt.Println("Showcase App running on:", app.currentRoute)
	select {}
}
