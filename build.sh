#!/bin/sh
# Build the demo into docs/, which is what GitHub Pages serves and what
# embed.go embeds.
#
# Both toolchains are carried. TinyGo is at the root because the binary is a
# fraction of the size and is fetched before anything appears; the standard Go
# build is a click away in the page header, because TinyGo occasionally
# miscompiles something and having the other one to hand is how you find out
# that is what happened.
#
#   ./build.sh          both
#   ./build.sh tinygo   TinyGo only
#   ./build.sh go       standard Go only
set -eu

cd "$(dirname "$0")"

# The two toolchains ship different loader shims and they are not
# interchangeable: the Go one against a TinyGo module fails at instantiation
# with an import it cannot satisfy.
build_tinygo() {
	mkdir -p docs/tinygo
	# -stack-size: the default wasm stack overflows in regexp/syntax.Simplify,
	#   which recurses once per repetition when expanding a bounded repeat like
	#   the {1,256} in goldmark's linkify pattern. The overflow does not fault,
	#   it corrupts: Simplify returns an unexpanded OpRepeat and regexp's
	#   compiler then reaches a case it treats as unreachable, panicking at
	#   startup with what reads as a syntax error.
	# -interp-timeout: the compile-time interpreter gives up on goldmark at the
	#   default three minutes and the build fails outright.
	#
	# This build still does not work. Its linear memory is 3.3GB at load,
	# before any Go runs — declared, not grown — and past 2GB TinyGo's shim
	# hands JavaScript negative addresses, so every DataView write throws and
	# the terminals stay blank. -gc=precise was tried and changed nothing,
	# which is what you would expect for memory that was never allocated by
	# the collector. The standard Go build of the same source declares 29MB.
	tinygo build -o docs/tinygo/desktop.wasm -target wasm -no-debug \
		-stack-size 1MB -interp-timeout 10m ./cmd/desktop
	cp "$(tinygo env TINYGOROOT)/targets/wasm_exec.js" docs/tinygo/wasm_exec.js
}

build_go() {
	mkdir -p docs
	GOOS=js GOARCH=wasm go build -o docs/desktop.wasm ./cmd/desktop
	cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" docs/wasm_exec.js
}

case "${1:-both}" in
both)   build_tinygo; build_go ;;
tinygo) build_tinygo ;;
go)     build_go ;;
*)      echo "usage: $0 [both|tinygo|go]" >&2; exit 2 ;;
esac

ls -la docs/desktop.wasm docs/tinygo/desktop.wasm 2>/dev/null || true
