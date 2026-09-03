# shims

Files that belong in someone else's module.

Each is the whole of what one library needs to build for `js/wasm`. They are
kept here because a Go module cannot add a file to another module: the only
ways to apply one are a fork imported by its own path, or a `replace`. None of
them are patches — every one is a new file, adding the `js && wasm` case to a
set of build-tagged platform files that already has a `_unix` and a `_windows`.

That shape is the common one. A library asks the operating system for a tty,
splits the answer per platform, and simply has no branch for a platform where
the question does not apply. Filling it in is short because in a browser the
answer is always "there isn't one".

## atomicgo-keyboard — forked

For `atomicgo.dev/keyboard`, which `pterm` imports for its interactive
components. Four symbols: `initInput`, `restoreInput`, `closeInput`,
`openInputTTY`.

Without it `pterm` does not compile for `js/wasm` at all, even for the parts
that only print — importing the root package is enough to pull the tty code in.

```
github.com/0magnet/keyboard        # fork of atomicgo.dev/keyboard
  + tty_js.go                      # this file
```

Then, in the consumer:

```
replace atomicgo.dev/keyboard => github.com/0magnet/keyboard v0.2.9-…
```

A `replace` is unavoidable here: `pterm` imports `atomicgo.dev/keyboard` by
that path from inside its own source, so nothing but a replacement can
redirect it. The alternative is upstream taking the file.

## bubbletea

Not shipped here yet, but the same four-line story, and there is already a
fork to put it in — `0magnet/bubbletea`. The symbols are `listenForResize`,
`initInput`, `suspendSupported` and `suspendProcess`; v1 also wants
`openInputTTY`. Verified to build with nothing else changed:

```go
//go:build js && wasm

package tea

func (p *Program) listenForResize(done chan struct{}) { defer close(done); <-p.ctx.Done() }
func (p *Program) initInput() error                   { return nil }

const suspendSupported = false

func suspendProcess() {}
```

## atotto/clipboard — forked

What `bubbles` and `huh` block on, through `textinput`. It has
`clipboard_darwin.go`, `_plan9`, `_unix` and `_windows`, and no fallback, so
`js/wasm` finds no `readAll`/`writeAll`.
