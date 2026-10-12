// Package commands is the tuiwasm command line: serve the built demo, or run
// a demo in a real terminal.
package commands

import (
	"github.com/0magnet/calvin/clihelp"
	"github.com/spf13/cobra"
)

// RootCmd is the root command
var RootCmd = &cobra.Command{
	Use:                   "tuiwasm",
	Short:                 "serve the tuiwasm demo, or run a demo in a terminal",
	Long:                  "serve the tuiwasm demo, or run a demo in a terminal",
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
}

// Execute executes the root command
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
}

// Init styles the command tree. Call it once subcommands are attached.
func Init() { clihelp.Init(RootCmd, "tuiwasm") }
