//go:build js && wasm

// Command showcase runs one of the demos inside an xterm-go terminal.
//
// This is the composition root and the only place that knows about both the
// terminal and the demos. Pick one with ?demo=name; without that it lists
// what there is.
//
// Text demos are handed an io.Writer and never see tcell — most libraries
// only write styled text, and for those the terminal alone is the whole
// runtime. Screen demos get a tcell.Screen through the adapter.
package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/0magnet/tuiwasm/demos"
	"github.com/0magnet/tuiwasm/play"
	"github.com/0magnet/tuiwasm/xwrite"

	// Importing a demo registers it. Adding one to this list is the whole of
	// wiring it up.
	_ "github.com/0magnet/tuiwasm/demos/anim"
	_ "github.com/0magnet/tuiwasm/demos/charts"
	_ "github.com/0magnet/tuiwasm/demos/markdown"
	_ "github.com/0magnet/tuiwasm/demos/styles"
	_ "github.com/0magnet/tuiwasm/demos/tables"
	_ "github.com/0magnet/tuiwasm/demos/upstream/boxes"
	_ "github.com/0magnet/tuiwasm/demos/upstream/colors"
	_ "github.com/0magnet/tuiwasm/demos/upstream/unicode"
)

func main() {
	container := js.Global().Get("document").Call("getElementById", "terminal")
	if !container.Truthy() {
		fail("no #terminal element on the page")
		return
	}

	// play.NewTerminal rather than building one here: it sets the options this
	// needs and, more importantly, moves the terminal onto the WebGL renderer.
	// Half the demos are animations that repaint every cell every frame, and
	// the DOM renderer rebuilds a span per cell — on that path the page stops
	// answering rather than merely running slowly.
	term := play.NewTerminal(container)
	term.Focus()

	w := xwrite.New(term)

	d, ok := demos.Lookup(wanted())
	if !ok {
		menu(w)
		return
	}

	// play does the rest: it works out which shape the demo is, bridges a
	// tcell screen to the terminal when it needs one, and gives it a goroutine
	// because PollEvent blocks and the page has one thread.
	if _, err := play.In(d, term, container); err != nil {
		fail(err.Error())
		return
	}
	if d.Text != nil {
		fmt.Fprint(w, "\r\n\x1b[2m— done. change ?demo= to run another —\x1b[0m\r\n") //nolint:errcheck
	}

	// A wasm module whose main returns is torn down, taking the terminal
	// with it, so the page parks here whichever demo ran.
	select {}
}

// wanted reads ?demo= from the page's URL.
func wanted() string {
	loc := js.Global().Get("location")
	if !loc.Truthy() {
		return ""
	}
	params := js.Global().Get("URLSearchParams").New(loc.Get("search"))
	if !params.Truthy() {
		return ""
	}
	if v := params.Call("get", "demo"); v.Type() == js.TypeString {
		return v.String()
	}
	return ""
}

func menu(w *xwrite.Writer) {
	fmt.Fprint(w, "\x1b[1;36mtuiwasm\x1b[0m — Go terminal libraries in the browser\r\n") //nolint:errcheck
	fmt.Fprint(w, "\x1b[2mthe terminal is xterm-go, compiled to wasm\x1b[0m\r\n\r\n")    //nolint:errcheck
	for _, d := range demos.All() {
		kind := "text"
		if d.Screen != nil {
			kind = "tcell"
		}
		fmt.Fprintf(w, "  \x1b[1m%-10s\x1b[0m \x1b[2m%-6s\x1b[0m %s\r\n", d.Name, kind, d.Desc) //nolint:errcheck
	}
	fmt.Fprintf(w, "\r\nopen with \x1b[1m?demo=%s\x1b[0m\r\n", //nolint:errcheck
		strings.Join(names(), "\x1b[0m or \x1b[1m?demo="))
}

func names() []string {
	all := demos.All()
	out := make([]string, 0, len(all))
	for _, d := range all {
		out = append(out, d.Name)
	}
	return out
}

func fail(msg string) {
	js.Global().Get("console").Call("error", "showcase: "+msg)
}
