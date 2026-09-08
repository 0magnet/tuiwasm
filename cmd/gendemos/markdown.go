//go:build tuimarkdown

// Package main cmd/gendemos/markdown.go
//
// The markdown demo's registration, behind the same tag cmd/desktop puts it
// behind.
//
// Without this the table listed a demo the shipped binary does not contain:
// gendemos imported demos/markdown unconditionally and saw it register, while
// cmd/desktop imports it only under -tags tuimarkdown, so every published
// ?demo=markdown link answered "no demo called markdown". pterm never had the
// problem because its build tag is on the demo package itself, which makes it
// unimportable rather than merely unimported.
//
// The rule this restores: gendemos must be built the same way as the thing it
// documents, so the table cannot describe a build nobody ships.
package main

import _ "github.com/0magnet/tuiwasm/demos/markdown"
