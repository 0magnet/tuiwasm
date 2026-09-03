//go:build pterm

// Package pterm shows pterm's printers running in a browser.
//
// Built only with -tags pterm, because it needs a replace directive that the
// rest of the repo should not carry:
//
//	replace atomicgo.dev/keyboard => github.com/0magnet/keyboard v…
//
// pterm imports atomicgo.dev/keyboard by that path from inside its own
// source, so a fork alone cannot redirect it. The fork needs one new file,
// which is in shims/atomicgo-keyboard. Four functions, all of them empty:
// there is no tty to raw-mode in a page.
//
// Only the printers are used here. pterm's interactive prompts want keystrokes
// through that same keyboard package, and the shim makes it compile rather
// than makes it work — those need wiring to DOM events, the way xtcell does
// for tcell.
package pterm

import (
	"io"

	"github.com/pterm/pterm"

	"github.com/0magnet/tuiwasm/demos"
)

func init() {
	demos.Register(demos.Demo{
		Name: "pterm",
		Desc: "pterm — headers, bullets, boxes and bars",
		Text: run,
	})
}

func run(w io.Writer, cols, _ int) error {
	// pterm writes to its own package-level default, so it has to be pointed
	// at the terminal rather than handed a writer per call.
	pterm.SetDefaultOutput(w)

	// Color is detected through the tty, which under js/wasm reports that
	// there isn't one, so it has to be asserted or everything prints plain.
	pterm.EnableColor()

	if cols > 0 {
		pterm.DefaultCenter.CenterEachLineSeparately = true
	}

	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("pterm, compiled to WebAssembly")

	pterm.Println()

	pterm.DefaultSection.Println("What it took")

	pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "one new file in atomicgo.dev/keyboard"},
		{Level: 1, Text: "initInput, restoreInput, closeInput, openInputTTY"},
		{Level: 1, Text: "every one of them empty"},
		{Level: 0, Text: "a replace directive, because pterm imports it by path"},
		{Level: 0, Text: "EnableColor, since there is no tty to detect"},
	}).Render()

	pterm.DefaultBox.
		WithTitle("why it compiles at all").
		WithTitleBottomCenter().
		Println("The printers never needed a terminal.\nOnly the prompts did.")

	pterm.Println()

	bars := []pterm.Bar{
		{Label: "works as-is", Value: 19},
		{Label: "one file", Value: 6},
		{Label: "termios-bound", Value: 7},
		{Label: "impossible", Value: 1},
	}
	if err := pterm.DefaultBarChart.WithBars(bars).WithHorizontal().WithShowValue().Render(); err != nil {
		return err
	}

	pterm.Info.Println("counts from the compile probe, not from memory")
	return nil
}
