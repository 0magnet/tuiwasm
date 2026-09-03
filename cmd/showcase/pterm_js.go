//go:build js && wasm && pterm

package main

// Registered only with -tags pterm, which also needs the replace directive
// described in shims/README.md.
import _ "github.com/0magnet/tuiwasm/demos/pterm"
