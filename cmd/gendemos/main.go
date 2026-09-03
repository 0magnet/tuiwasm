// Command gendemos writes the README's table of demos from the registry.
//
//	gendemos            # rewrite README.md in place
//	gendemos -check     # exit 1 if it is out of date, changing nothing
//
// The region between the markers is replaced wholesale, the way gendocs.sh
// rewrites the dependency graph — a patched section drifts, a replaced one
// cannot:
//
//	<!-- BEGIN DEMOS --> … <!-- END DEMOS -->
//
// WHY THIS IS GENERATED. The table was written by hand and listed seven demos
// when there were twenty-nine: every animation added over a week of work was
// missing from it, and so was every link to one. A list of what exists, kept by
// hand beside the thing that actually knows, is a list that is wrong.
//
// It reads the registry the same way the page does — by importing the demo
// packages for their registration — so the table cannot name a demo that is not
// there or miss one that is.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/0magnet/tuiwasm/demos"

	// Importing a demo registers it. This is the same list cmd/desktop has, and
	// the reason the two are separate is that this one runs on the host: the
	// registry and the demos are ordinary packages, and only the terminal they
	// eventually draw into is js/wasm.
	_ "github.com/0magnet/tuiwasm/demos/anim"
	_ "github.com/0magnet/tuiwasm/demos/charts"
	_ "github.com/0magnet/tuiwasm/demos/markdown"
	_ "github.com/0magnet/tuiwasm/demos/proxima"
	_ "github.com/0magnet/tuiwasm/demos/styles"
	_ "github.com/0magnet/tuiwasm/demos/tables"
	_ "github.com/0magnet/tuiwasm/demos/upstream/boxes"
	_ "github.com/0magnet/tuiwasm/demos/upstream/colors"
	_ "github.com/0magnet/tuiwasm/demos/upstream/unicode"
	_ "github.com/0magnet/tuiwasm/demos/widgets"
)

const (
	begin = "<!-- BEGIN DEMOS -->"
	end   = "<!-- END DEMOS -->"

	// site is where the published page lives. Every demo is that one page and
	// that one .wasm with a different query string; there is no per-demo build.
	site = "https://0magnet.github.io/tuiwasm/"
)

var (
	path  = flag.String("readme", "README.md", "the file to rewrite")
	check = flag.Bool("check", false, "report whether the file is up to date; write nothing")
)

func main() {
	flag.Parse()

	old, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gendemos:", err)
		os.Exit(1)
	}
	next, err := replace(string(old), table())
	if err != nil {
		fmt.Fprintln(os.Stderr, "gendemos:", err)
		os.Exit(1)
	}

	if *check {
		if next != string(old) {
			fmt.Fprintln(os.Stderr, "gendemos: out of date — run `go run ./cmd/gendemos`")
			os.Exit(1)
		}
		fmt.Println("gendemos: up to date")
		return
	}
	if next == string(old) {
		fmt.Println("gendemos: no change")
		return
	}
	if err := os.WriteFile(*path, []byte(next), 0o644); err != nil { //nolint:gosec // a README is world-readable by design
		fmt.Fprintln(os.Stderr, "gendemos:", err)
		os.Exit(1)
	}
	fmt.Printf("gendemos: wrote %d demos to %s\n", len(demos.All()), *path)
}

// table renders the demo list.
func table() string {
	var b strings.Builder
	b.WriteString("| demo | shape | what it is |\n| --- | --- | --- |\n")
	for _, d := range demos.All() {
		shape := "screen"
		if d.Text != nil {
			shape = "text"
		}
		fmt.Fprintf(&b, "| [`%s`](%s?demo=%s) | %s | %s |\n", d.Name, site, d.Name, shape, d.Desc)
	}
	return b.String()
}

// replace swaps what is between the markers, leaving them in place.
func replace(doc, body string) (string, error) {
	i := strings.Index(doc, begin)
	j := strings.Index(doc, end)
	if i < 0 || j < 0 || j < i {
		return "", fmt.Errorf("markers %s … %s not found in the document", begin, end)
	}
	var out bytes.Buffer
	out.WriteString(doc[:i+len(begin)])
	out.WriteString("\n")
	out.WriteString(body)
	out.WriteString(doc[j:])
	return out.String(), nil
}
