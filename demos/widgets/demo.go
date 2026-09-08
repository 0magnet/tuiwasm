// Package widgets registers a tview demo: lists, tables, forms and text
// views laid out by a flexbox, running in a browser tab.
//
// # WHY THIS IS ON TCELL V2 WHEN EVERYTHING ELSE MOVED TO V3
//
// It was here once and was deleted. Commit e1cc626, "Follow the dependencies
// to tcell v3, and drop the tview demo with them", took it out because tview
// is staying on v2 — rivo/tview#1145 asked for the move and the maintainer
// declined on compatibility grounds — and the reasoning recorded at the time
// was that a module gets one tcell.
//
// That turned out not to be true of this module. v2 and v3 are different
// module paths, so both can be required at once, and xtcell2 — the
// pre-migration driver, kept — already paints a v2 program into an xterm-go
// terminal for the proxima2 demo. So the thing that made this impossible had
// stopped being true, and nobody had come back for it.
//
// It runs on demos.Demo.ScreenV2 for that reason, alongside proxima2, while
// everything written here runs on v3.
//
// # THE ONE TRAP
//
// tview.Application.SetScreen calls Init on the screen it is given, right
// then, when Run has not started yet. tcell's web screen hardcodes 80x24 in
// Init, so the size the host worked out is thrown away and the layout is
// computed for a window that is not the one on screen — which looks like
// tview being broken rather than like a size being reset. SetSize goes back
// afterwards, which is what demos.Demo.Screen documents and what this does.
package widgets

import (
	"fmt"
	"runtime/debug"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/0magnet/tuiwasm/demos"
)

func init() {
	demos.Register(demos.Demo{
		Name:     "widgets",
		Desc:     "tview — a flexbox of lists, tables and text views on tcell v2",
		ScreenV2: run,
	})
}

// The widgets the list offers, and what the pane says about each. Kept
// together so the list and the description cannot drift apart.
var pages = []struct {
	name, body string
}{
	{"Flex", "Everything here is inside a Flex.\n\n" +
		"tview lays out with flexbox rather than absolute\n" +
		"coordinates, so a pane keeps its share of the\n" +
		"window when the terminal is resized — try it, the\n" +
		"browser window works.\n\n" +
		"Rows hold columns hold rows, as deep as you like."},
	{"List", "The pane on the left is a List.\n\n" +
		"Move with the arrow keys or j and k. Each item can\n" +
		"carry a shortcut rune, a secondary line, and a\n" +
		"callback fired on selection — which is what is\n" +
		"redrawing this text."},
	{"Table", "Below is a Table with a fixed header row.\n\n" +
		"It scrolls independently of everything else and can\n" +
		"select by cell, by row, or not at all. The header\n" +
		"stays put because the first row is fixed."},
	{"TextView", "This pane is a TextView.\n\n" +
		"It wraps, scrolls, and understands its own color\n" +
		"tags — so [red]this[-] is red without touching a\n" +
		"single escape sequence yourself."},
	{"Borders", "Every pane draws its own border and title.\n\n" +
		"The line-drawing glyphs come from the terminal font,\n" +
		"which here is xterm-go rendering with WebGL. What\n" +
		"tview thinks it is talking to is a tcell.Screen; it\n" +
		"has no idea it is in a browser."},
}

func run(screen tcell.Screen, cols, rows int) error {
	app := tview.NewApplication().SetScreen(screen)

	// SetScreen has just called Init, which put the screen back to tcell's
	// hardcoded 80x24. See the package comment.
	screen.SetSize(cols, rows)

	body := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	body.SetBorder(true).SetTitle(" TextView ")

	list := tview.NewList().ShowSecondaryText(false)
	for i, p := range pages {
		i := i
		list.AddItem(p.name, "", rune('1'+i), func() {
			body.SetText(pages[i].body)
		})
	}
	list.SetChangedFunc(func(i int, _, _ string, _ rune) {
		if i >= 0 && i < len(pages) {
			body.SetText(pages[i].body)
		}
	})
	list.SetBorder(true).SetTitle(" List ")
	body.SetText(pages[0].body)

	table := tview.NewTable().SetFixed(1, 0).SetSelectable(true, false)
	head := []string{"widget", "tview type", "what it is for"}
	for c, h := range head {
		table.SetCell(0, c, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1))
	}
	rowsOf := [][3]string{
		{"list", "*tview.List", "a column of choices"},
		{"table", "*tview.Table", "rows and columns, scrollable"},
		{"text", "*tview.TextView", "wrapped text with color tags"},
		{"input", "*tview.InputField", "one line of typed input"},
		{"form", "*tview.Form", "fields and buttons together"},
		{"tree", "*tview.TreeView", "a hierarchy that folds"},
		{"pages", "*tview.Pages", "one of several, swapped"},
	}
	for r, cells := range rowsOf {
		for c, s := range cells {
			table.SetCell(r+1, c, tview.NewTableCell(" "+s).SetExpansion(1))
		}
	}
	table.SetBorder(true).SetTitle(" Table ")

	status := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	status.SetText(fmt.Sprintf(
		"[yellow]tab[-] moves between panes   [yellow]arrows[-] move within one   [yellow]q[-] quits"+
			"   ·   tview %s on tcell v2, in %dx%d", tviewVersion(), cols, rows))

	// The status line is a header and not a footer, which is not where a
	// status line belongs.
	//
	// The last row of the terminal is not visible in this host: the identical
	// TextView, with the identical text, renders correctly as the first item
	// of this Flex and renders nothing at all as the last one. Something
	// between AutoFit and the container is an off-by-one, so the bottom row is
	// clipped. Until that is chased down, a demo that puts anything on the
	// last row is a demo with an invisible line in it.
	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(body, 0, 3, false).
		AddItem(table, 0, 4, false)

	main := tview.NewFlex().
		AddItem(list, 22, 0, true).
		AddItem(right, 0, 1, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(status, 1, 0, false).
		AddItem(main, 0, 1, true)

	// Tab cycles the focus, q leaves. Escape is left alone: tview gives it to
	// the focused widget, and a List uses it.
	focus := []tview.Primitive{list, table, body}
	at := 0
	app.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyTab:
			at = (at + 1) % len(focus)
			app.SetFocus(focus[at])
			return nil
		case ev.Rune() == 'q':
			app.Stop()
			return nil
		}
		return ev
	})

	return app.SetRoot(root, true).Run()
}

// tviewVersion is the tview release this was built against, read from the
// build info rather than written down, so the status line cannot claim a
// version the binary does not contain.
func tviewVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "?"
	}
	for _, d := range info.Deps {
		if d.Path == "github.com/rivo/tview" {
			return d.Version
		}
	}
	return "?"
}
