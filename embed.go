// Package tuiwasm carries the built demo: the page, both wasm builds and
// their loader shims.
//
// The package doc lives here rather than in a js/wasm-only file so that this,
// and therefore the serve command, is available to a native binary — which is
// the whole point of embedding the build.
//
// Embedding means the server is one binary with nothing to host and nothing
// to fetch: it works on a machine with no network and no checkout. The cost
// is that the binary carries both wasm builds, and that a build has to have
// been run before this package compiles, which is why docs/ holds a committed
// build rather than being a build artifact.
//
// Nothing is paid for by importers who do not serve it. Go drops an embedded
// file set that nothing reads, so a program importing this package for
// anything else does not carry the wasm.
package tuiwasm

import (
	"embed"
	"io/fs"
)

// docs is the built site, laid out the way GitHub Pages will serve it: the
// TinyGo build at the root, the standard Go build one level down, so the
// smaller one is what a visitor is given and the other is a link away.
//
//go:embed docs
var docs embed.FS

// Assets returns the built site, ready for http.FileServerFS.
func Assets() fs.FS {
	sub, err := fs.Sub(docs, "docs")
	if err != nil {
		// docs is embedded above, so this cannot fail at runtime. If it does,
		// the package is broken rather than the caller.
		panic("tuiwasm: embedded assets missing: " + err.Error())
	}
	return sub
}
