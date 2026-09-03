// Package widgets shows tview running through the tcell adapter.
//
// tview needed nothing: it is written against tcell.Screen, so making tcell
// work in a browser made tview work too. That is the argument for adapting
// tcell rather than any of the higher-level libraries — termdash arrives on
// the same ticket.
package widgets

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/0magnet/tuiwasm/demos"
)

func init() {
	demos.Register(demos.Demo{
		Name:   "widgets",
		Desc:   "tview — list, form and table, through the tcell adapter",
		Screen: run,
	})
}

func run(s tcell.Screen, cols, rows int) error {
	app := tview.NewApplication()

	// SetScreen calls Init on the screen it is given, which puts tcell's
	// hardcoded 80x24 back. The real size has to be restored after.
	app.SetScreen(s)
	if cols > 0 && rows > 0 {
		s.SetSize(cols, rows)
	}

	status := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[::d]tab moves between panes · q quits[-:-:-]")

	list := tview.NewList().
		AddItem("tcell", "the screen underneath", 0, nil).
		AddItem("tview", "these widgets", 0, nil).
		AddItem("termdash", "also tcell", 0, nil).
		AddItem("lipgloss", "no screen needed", 0, nil)
	list.SetBorder(true).SetTitle(" built on tcell ")

	form := tview.NewForm().
		AddInputField("name", "", 20, nil, nil).
		AddDropDown("renderer", []string{"webgl", "dom"}, 0, nil).
		AddCheckbox("mouse", false, nil)
	form.SetBorder(true).SetTitle(" a form, in wasm ")

	table := tview.NewTable().SetBorders(false)
	hdr := []string{"library", "js/wasm"}
	for c, h := range hdr {
		table.SetCell(0, c, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).SetSelectable(false))
	}
	for r, row := range [][2]string{
		{"tcell", "yes"}, {"tview", "yes"}, {"termdash", "yes"},
		{"bubbletea", "one file"}, {"termbox", "no"},
	} {
		for c, v := range row {
			table.SetCell(r+1, c, tview.NewTableCell(fmt.Sprintf(" %s ", v)))
		}
	}
	table.SetBorder(true).SetTitle(" compatibility ")

	body := tview.NewFlex().
		AddItem(list, 0, 1, true).
		AddItem(form, 0, 1, false).
		AddItem(table, 0, 1, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(body, 0, 1, true).
		AddItem(status, 1, 0, false)

	focus := []tview.Primitive{list, form, table}
	at := 0
	app.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyTab:
			at = (at + 1) % len(focus)
			app.SetFocus(focus[at])
			return nil
		case ev.Rune() == 'q', ev.Key() == tcell.KeyEscape, ev.Key() == tcell.KeyCtrlC:
			app.Stop()
			return nil
		}
		return ev
	})

	return app.SetRoot(root, true).Run()
}
