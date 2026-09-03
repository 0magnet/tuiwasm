// Command tui runs a demo in a real terminal.
//
// The demos are written against tcell.Screen or an io.Writer and know nothing
// about the browser, so the same code that draws into a page draws into a
// terminal. This is the command that proves it, and it is the one to reach for
// when comparing behavior — key repeat while a key is held, for instance,
// which a browser reports on its own schedule rather than the terminal's.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/0magnet/calvin"
	cc "github.com/0magnet/coloredcobra"
	tcell2 "github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/0magnet/tuiwasm/demos"

	// Importing a demo registers it, exactly as the wasm builds do.
	_ "github.com/0magnet/tuiwasm/demos/anim"
	_ "github.com/0magnet/tuiwasm/demos/charts"
	_ "github.com/0magnet/tuiwasm/demos/markdown"
	_ "github.com/0magnet/tuiwasm/demos/proxima"
	_ "github.com/0magnet/tuiwasm/demos/proxima2"
	_ "github.com/0magnet/tuiwasm/demos/styles"
	_ "github.com/0magnet/tuiwasm/demos/tables"
	_ "github.com/0magnet/tuiwasm/demos/upstream/boxes"
	_ "github.com/0magnet/tuiwasm/demos/upstream/colors"
	_ "github.com/0magnet/tuiwasm/demos/upstream/unicode"
)

var list bool

func init() {
	RootCmd.Flags().BoolVarP(&list, "list", "l", false, "list the demos and exit")
	var helpflag bool
	RootCmd.SetUsageTemplate(help)
	RootCmd.PersistentFlags().BoolVarP(&helpflag, "help", "h", false, "help for "+RootCmd.Use)
	RootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	RootCmd.PersistentFlags().MarkHidden("help") //nolint
}

// RootCmd is the root command
var RootCmd = &cobra.Command{
	Use:                   "tui [demo]",
	Short:                 "run a demo in this terminal",
	Long:                  calvin.AsciiFont("tui") + "\nrun a demo in this terminal",
	Args:                  cobra.MaximumNArgs(1),
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(_ *cobra.Command, args []string) error {
		if list || len(args) == 0 {
			listDemos()
			return nil
		}
		d, ok := demos.Lookup(args[0])
		if !ok {
			return fmt.Errorf("no demo %q; try --list", args[0])
		}
		if d.Screen != nil {
			return runScreen(d)
		}
		return runText(d)
	},
}

func listDemos() {
	all := demos.All()
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	fmt.Println("demos:")
	for _, d := range all {
		kind := "text"
		if d.Screen != nil {
			kind = "tcell"
		}
		if d.ScreenV2 != nil {
			kind = "tcell2"
		}
		fmt.Printf("  %-10s %-6s %s\n", d.Name, kind, d.Desc)
	}
	fmt.Println("\nrun one with: tui <name>")
}

// runScreen gives the demo a real terminal screen.
//
// Fini runs however the demo leaves: a panic that skips it puts the terminal
// into raw mode with the alternate screen still up, which looks like the
// shell has broken.
func runScreen(d demos.Demo) (err error) {
	// A tcell v2 demo gets a real tcell/v2 screen; in a terminal both majors
	// drive the same tty, so the one difference is which package makes it.
	if d.ScreenV2 != nil {
		screen, err := tcell2.NewScreen()
		if err != nil {
			return err
		}
		if err := screen.Init(); err != nil {
			return err
		}
		defer screen.Fini()

		cols, rows := screen.Size()
		return d.ScreenV2(screen, cols, rows)
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	cols, rows := screen.Size()
	return d.Screen(screen, cols, rows)
}

// runText writes the demo to stdout, fitted to the terminal.
func runText(d demos.Demo) error {
	cols, rows := 80, 24
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		cols, rows = w, h
	}
	return d.Text(os.Stdout, cols, rows)
}

func main() {
	cc.Init(&cc.Config{
		RootCmd:         RootCmd,
		Headings:        cc.HiBlue + cc.Bold,
		Commands:        cc.HiBlue + cc.Bold,
		CmdShortDescr:   cc.HiBlue,
		Example:         cc.HiBlue + cc.Italic,
		ExecName:        cc.HiBlue + cc.Bold,
		Flags:           cc.HiBlue + cc.Bold,
		FlagsDescr:      cc.HiBlue,
		NoExtraNewlines: true,
		NoBottomNewline: true,
	})
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error()))
		os.Exit(1)
	}
}

const help = "{{if .HasAvailableSubCommands}}{{end}} {{if gt (len .Aliases) 0}}\r\n\r\n" +
	"{{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}" +
	"Available Commands:{{range .Commands}}  {{if and (ne .Name \"completion\") .IsAvailableCommand}}\r\n  " +
	"{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}\r\n\r\n" +
	"Flags:\r\n" +
	"{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}\r\n\r\n" +
	"Global Flags:\r\n" +
	"{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}\r\n\r\n"
