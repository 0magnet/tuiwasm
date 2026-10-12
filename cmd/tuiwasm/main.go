// Command tuiwasm serves the built demo (tuiwasm serve) or runs one of the
// demos in a real terminal (tuiwasm tui).
package main

import (
	"fmt"
	"os"

	"github.com/0magnet/tuiwasm/cmd/tuiwasm/commands"
)

func main() {
	commands.Init()
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
