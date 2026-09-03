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
	# TinyGo trails each new Go release by some weeks; until it catches up, a
	# build against the newer Go fails outright. The helper reports the newest
	# Go this TinyGo accepts, or "auto" once the system one will do.
	GOTOOLCHAIN=$(sh scripts/tinygo-toolchain.sh); export GOTOOLCHAIN
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
	# This build works. It did not for a long time, and the reason is worth
	# keeping, because the diagnosis was wrong twice before it was right.
	#
	# It used to grow linear memory to 3.44GB during package initialization —
	# grown, not declared: the module's memory section asks for 55 pages, 3.6MB.
	# Every one of those bytes came from a per-cell js.Value.String() in xtcell's
	# old drawCell, ten exact doublings of the heap, and the rewrite of that path
	# deleted the cause. It now settles at ~830MB, which is still a lot for four
	# terminals but is nowhere near a limit.
	#
	# What the 3.44GB tripped was a separate bug in TinyGo's shim: wasm32
	# pointers are i32, JavaScript sees an i32 as signed, and wasm_exec.js never
	# coerces one back, so the first pointer past 2GB arrived as -2146545368 —
	# 2148421928 unsigned, a byte well inside the buffer — and every DataView
	# write threw. Under 2GB it never fires, so this build no longer meets it.
	# That bug is real and still upstream's: tinygo-org/tinygo#5621.
	#
	# The other thing to know about the shim: sleepTicks leaked a setTimeout
	# chain per JS->Go callback, unbounded, which is tinygo-org/tinygo#5622.
	# Neither of those is fixed in a released TinyGo yet.
	#
	# -opt=2 is temporary. The default is -opt=z, which is what this build
	# wants once it works, but -opt=z is much slower to compile and the
	# build is being run repeatedly right now. Put -opt=z back — by dropping
	# the flag — once the behaviour above is settled.
	tinygo build -o docs/tinygo/desktop.wasm -target wasm -no-debug -opt=2 \
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
