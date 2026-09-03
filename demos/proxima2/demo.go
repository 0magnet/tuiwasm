// Package proxima2 registers Escape from Proxima 5 on tcell v2.
//
// It is the same game as the proxima demo, one tcell major back: the
// tcell-v2 lineage of the fork, published as module major version 2 of
// github.com/0magnet/proxima5. Running both side by side is the point —
// the two lineages of the same 2016 curses program, each against the
// browser driver for its own tcell major.
package proxima2

import (
	"strconv"

	"github.com/gdamore/tcell/v2"

	game "github.com/0magnet/proxima5/v2"

	"github.com/0magnet/tuiwasm/demos"
)

func init() {
	demos.Register(demos.Demo{
		Name:     "proxima2",
		Desc:     "Escape from Proxima 5 on tcell v2 — the same game, one major back",
		ScreenV2: run,
	})
}

func run(screen tcell.Screen, cols, rows int) error {
	// The game asks for at least 80x24 and is unplayable below it. Saying so
	// beats letting it draw off the edge of a small window.
	if cols < 80 || rows < 24 {
		return tooSmall(screen, cols, rows)
	}
	g := game.NewGame(screen)
	if err := g.Init(); err != nil {
		return err
	}
	return g.Run()
}

func tooSmall(screen tcell.Screen, cols, rows int) error {
	msg := "Proxima 5 needs at least 80x24 — this window is " +
		strconv.Itoa(cols) + "x" + strconv.Itoa(rows) + ". Make it bigger and reopen."
	st := tcell.StyleDefault.Foreground(tcell.ColorRed)
	for i, r := range []rune(msg) {
		if i < cols {
			screen.SetContent(i, 0, r, nil, st)
		}
	}
	screen.Show()
	return nil
}
