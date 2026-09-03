// Command serve serves the built demo from a single binary.
//
// The page, both wasm builds and their loader shims are embedded, so this
// needs no checkout, no network and nothing else installed. It exists because
// a wasm page cannot be opened with file:// — the browser refuses to
// instantiate a module fetched that way — so trying it locally otherwise
// means finding a static server and getting its MIME types right.
//
// It serves what GitHub Pages will serve, as Pages will serve it: same
// layout, same paths, and gzip on the wasm, so what you try locally is what a
// visitor gets.
package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/0magnet/calvin"
	cc "github.com/ivanpirog/coloredcobra"
	"github.com/spf13/cobra"

	"github.com/0magnet/tuiwasm"
)

var (
	addr   string
	open   bool
	nogzip bool
)

func init() {
	RootCmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1:8780", "address to listen on")
	RootCmd.Flags().BoolVarP(&open, "open", "o", true, "open a browser at the served page")
	RootCmd.Flags().BoolVar(&nogzip, "no-gzip", false, "serve uncompressed, to see what the wasm costs on the wire")
	var helpflag bool
	RootCmd.SetUsageTemplate(help)
	RootCmd.PersistentFlags().BoolVarP(&helpflag, "help", "h", false, "help for "+RootCmd.Use)
	RootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	RootCmd.PersistentFlags().MarkHidden("help") //nolint
}

// RootCmd is the root command
var RootCmd = &cobra.Command{
	Use:                   "serve",
	Short:                 "serve the built demo, as GitHub Pages would",
	Long:                  calvin.AsciiFont("tuiwasm") + "\nserve the built demo, as GitHub Pages would",
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Run: func(_ *cobra.Command, _ []string) {
		// Some systems have a registry entry mapping .wasm to something else,
		// and a wrong Content-Type makes instantiateStreaming refuse the
		// module with an error that says nothing about MIME types.
		if err := mime.AddExtensionType(".wasm", "application/wasm"); err != nil {
			log.Printf("serve: could not register the wasm MIME type: %v", err)
		}

		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("serve: %v", err)
		}
		url := "http://" + ln.Addr().String() + "/"
		if strings.HasPrefix(ln.Addr().String(), "[::]") || strings.HasPrefix(addr, ":") {
			url = fmt.Sprintf("http://localhost:%d/", ln.Addr().(*net.TCPAddr).Port)
		}

		h := http.FileServerFS(tuiwasm.Assets())
		if !nogzip {
			h = gzipping(h)
		}

		fmt.Printf("serving on %s\n", url)
		fmt.Printf("  /        TinyGo build\n  /go/     standard Go build\n")
		if open {
			go browse(url)
		}
		srv := &http.Server{Handler: noCache(h), ReadHeaderTimeout: 10 * time.Second}
		log.Fatal(srv.Serve(ln))
	},
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
		log.Fatal("Failed to execute command: ", err)
	}
}

// gzipping compresses responses the way GitHub Pages does.
//
// Pages serves application/wasm with content-encoding gzip without being
// asked, which takes this build from 22M to about 4M. Serving it plain here
// would make a local try feel several times heavier than the real thing, and
// would hide the number that actually matters.
func gzipping(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		// Length is the uncompressed size and would be wrong once encoded;
		// dropping it lets the response be chunked instead.
		w.Header().Del("Content-Length")
		gz := gzip.NewWriter(w)
		// The response has already gone out by the time this runs, so a
		// failed flush cannot be reported to anyone.
		defer gz.Close() //nolint:errcheck
		h.ServeHTTP(gzipWriter{ResponseWriter: w, w: gz}, r)
	})
}

type gzipWriter struct {
	http.ResponseWriter
	w io.Writer
}

func (g gzipWriter) Write(p []byte) (int, error) { return g.w.Write(p) }

// noCache keeps a rebuilt wasm from being served out of the browser cache,
// which otherwise makes a fresh build look like it changed nothing.
func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

func browse(url string) {
	time.Sleep(300 * time.Millisecond) // let the listener settle first
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	// The command is one of three constants chosen just above, and the URL is
	// the address this process is listening on — neither comes from outside.
	if err := exec.Command(cmd, append(args, url)...).Start(); err != nil { //nolint:gosec
		log.Printf("serve: could not open a browser (%v); visit %s", err, url)
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
