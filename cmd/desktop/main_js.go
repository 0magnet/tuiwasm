//go:build js && wasm

// Command desktop shows the demos together, each in its own window on a desk.
//
// This replaces picking one demo per page. Several terminals side by side is
// what the collection is actually for: the point is that these libraries all
// run in a browser, and that reads better as six windows than as six URLs.
package main

import (
	"strings"

	"syscall/js"

	"github.com/0magnet/desk"

	"github.com/0magnet/tuiwasm/demos"
	"github.com/0magnet/tuiwasm/deskapp"
	"github.com/0magnet/tuiwasm/play"

	// Importing a demo registers it, and deskapp turns whatever is registered
	// into a window. Adding one to this list is the whole of wiring it up.
	_ "github.com/0magnet/tuiwasm/demos/anim"
	_ "github.com/0magnet/tuiwasm/demos/charts"
	// demos/markdown is registered by markdown_js.go, behind the tuimarkdown
	// build tag: it reaches glamour, which brings goldmark and chroma with it.
	_ "github.com/0magnet/tuiwasm/demos/proxima"
	_ "github.com/0magnet/tuiwasm/demos/proxima2"
	_ "github.com/0magnet/tuiwasm/demos/styles"
	_ "github.com/0magnet/tuiwasm/demos/tables"
	_ "github.com/0magnet/tuiwasm/demos/upstream/boxes"
	_ "github.com/0magnet/tuiwasm/demos/upstream/colors"
	_ "github.com/0magnet/tuiwasm/demos/upstream/unicode"
)

// opened is what the desk starts with, and the order they tile in. The rest
// stay in the launcher: a desktop that opens every window it has is a wall,
// not a demonstration.
//
// Several tcell demos may now be open at once. They used to take turns —
// tcell's wasm screen reaches the page through global function names, so a page
// had room for exactly one and the others drew nothing — which is no longer how
// they are drawn; see xtcell. What still applies is that only the window with
// the keyboard advances its frames, because drawing one nobody is looking at is
// work for nothing, so an animation behind another holds its last frame.
var opened = []string{"shell", "styles", "tables", "boxes"}

// Windows are tiled rather than left to cascade. Cascading is the right
// default for a desktop, where you work in one window at a time, but here the
// whole point is seeing several libraries drawing at once — stacked, three of
// the four are just a title bar.
const (
	cols = 2
	gap  = 8.0
)

func main() {
	root := js.Global().Get("document").Call("getElementById", "desktop")

	// ?demo=name gives one demo the whole page instead of a window on the desk.
	//
	// Same page and same module: a demo is picked out of the registry at
	// startup, so a link to one costs nothing to serve and nothing to build.
	// Deep-linking a single animation is most of what anyone wants from this —
	// an animation is worth looking at full screen, and a tile is not.
	if name := query("demo"); name != "" {
		fullScreen(root, name)
		return
	}

	if root.Truthy() {
		desk.SetRoot(root)
	}
	listDemos()

	deskapp.RegisterAll()
	deskapp.RegisterShell()
	deskapp.RegisterFileApps()
	desk.NewPanel()

	// ?open=a,b,c overrides which windows the desk starts with, which is how
	// a demo that is not in the default four gets looked at without editing
	// this list — proxima wants a big window and does not belong in a tile.
	if v := query("open"); v != "" {
		opened = strings.Split(v, ",")
	}

	for i, name := range opened {
		opt := tile(root, i, len(opened))
		if _, err := desk.LaunchOpts(name, opt); err != nil {
			js.Global().Get("console").Call("error", "desktop: "+name+": "+err.Error())
		}
	}

	// A wasm module whose main returns is torn down, taking the desk with it.
	select {}
}

// fullScreen runs one demo with the page to itself.
//
// It never returns: a wasm module whose main returns is torn down, taking the
// terminal with it.
func fullScreen(root js.Value, name string) {
	if !root.Truthy() {
		return
	}
	d, ok := demos.Lookup(name)
	if !ok {
		root.Set("innerHTML", "")
		root.Set("textContent", "no demo called "+name)
		select {}
	}

	// Nothing around it. A page showing one animation should look like the
	// terminal it would run in, and a header and a back link are neither of
	// those — the browser already has a back button. The class takes the page
	// chrome away; see index.html.
	doc := js.Global().Get("document")
	if body := doc.Get("body"); body.Truthy() {
		body.Get("classList").Call("add", "solo")
	}
	doc.Set("title", d.Name)

	if _, err := play.Mount(d, root); err != nil {
		js.Global().Get("console").Call("error", "desktop: "+name+": "+err.Error())
	}
	select {}
}

// listDemos fills the header's list with a link to each demo's own page.
//
// Built from the registry rather than written out in the HTML, so a demo that
// is added gets a link by being registered and there is no second list to keep
// in step with the first.
func listDemos() {
	doc := js.Global().Get("document")
	nav := doc.Call("getElementById", "demolinks")
	if !nav.Truthy() {
		return
	}
	var b strings.Builder
	for _, d := range demos.All() {
		b.WriteString(`<a href="?demo=` + d.Name + `" title="` + d.Desc + `">` + d.Name + `</a> `)
	}
	nav.Set("innerHTML", b.String())
}

// tile returns the placement for window i of n, in a grid across root.
//
// A zero Options is returned when the desk has no measurable size yet, which
// leaves the placement to desk rather than asking for a window of nothing.
func tile(root js.Value, i, n int) desk.Options {
	if !root.Truthy() {
		return desk.Options{}
	}
	w := root.Get("clientWidth").Float()
	h := root.Get("clientHeight").Float()
	if w < 320 || h < 240 {
		return desk.Options{}
	}

	// Never more columns than there are windows, or a lone window gets half
	// the desk and the other half stays empty — which for a demo with a
	// minimum width is the difference between running and refusing to.
	nc := cols
	if n < nc {
		nc = n
	}
	rows := (n + nc - 1) / nc
	cw := (w - gap*float64(nc+1)) / float64(nc)
	ch := (h - gap*float64(rows+1)) / float64(rows)

	c, r := i%nc, i/nc
	return desk.Options{
		X:      gap + float64(c)*(cw+gap),
		Y:      gap + float64(r)*(ch+gap),
		Width:  cw,
		Height: ch,
	}
}

// query reads one parameter from the page's URL.
func query(name string) string {
	loc := js.Global().Get("location")
	if !loc.Truthy() {
		return ""
	}
	p := js.Global().Get("URLSearchParams").New(loc.Get("search"))
	if !p.Truthy() {
		return ""
	}
	if v := p.Call("get", name); v.Type() == js.TypeString {
		return v.String()
	}
	return ""
}
