//go:build js && wasm

// This file belongs in a fork of atomicgo.dev/keyboard, alongside its
// tty_unix.go and tty_windows.go. See ../README.md.
//
// pterm imports this package for its interactive components, and importing
// pterm's root package is enough to pull it in — so without these four
// symbols pterm does not build for js/wasm even to print a table.

package keyboard

import (
	"errors"
	"os"
)

// There is no tty in a browser and nothing to put into raw mode: keystrokes
// arrive as DOM events, already decoded. So there is no state to save and
// none to restore.
func initInput() error { return nil }

func restoreInput() error { return nil }

func closeInput() {}

// openInputTTY has no answer here. Returning an error rather than a nil file
// keeps the failure at the call that wanted a tty.
func openInputTTY() (*os.File, error) {
	return nil, errors.New("keyboard: no tty in a browser")
}
