//go:build js && wasm && tuimarkdown

// Package main cmd/desktop/markdown_js.go
// The markdown demo, kept out of the default build.
//
// It is the single heaviest thing in the tree: glamour pulls goldmark for
// parsing and chroma for highlighting the fenced code block, which is 23 of the
// wasm's packages and a large share of a TinyGo compile. Everything else here
// is a terminal widget. Build with -tags=tuimarkdown to include it.
package main

import _ "github.com/0magnet/tuiwasm/demos/markdown"
