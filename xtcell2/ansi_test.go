package xtcell2

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// The sequences are the whole of what this package does, and none of them
// need a browser to check. Everything in this file builds and runs natively.

func TestCUPIsOneBased(t *testing.T) {
	// tcell's origin is 0,0 and ANSI's is 1,1. An off-by-one here puts the
	// whole screen one row up and one column left, which looks like a border
	// problem rather than an addressing one.
	if got, want := cup(0, 0), "\x1b[1;1H"; got != want {
		t.Errorf("cup(0,0) = %q, want %q", got, want)
	}
	if got, want := cup(9, 4), "\x1b[5;10H"; got != want {
		t.Errorf("cup(9,4) = %q, want %q", got, want)
	}
}

func TestSGRTruecolor(t *testing.T) {
	got := sgr(0xff8000, 0x102030, tcell.AttrNone, tcell.UnderlineStyleNone)
	want := "\x1b[0;38;2;255;128;0;48;2;16;32;48m"
	if got != want {
		t.Errorf("sgr = %q, want %q", got, want)
	}
}

func TestSGRAttributes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		attr  tcell.AttrMask
		param string
	}{
		{"bold", tcell.AttrBold, "1"},
		{"dim", tcell.AttrDim, "2"},
		{"italic", tcell.AttrItalic, "3"},
		{"blink", tcell.AttrBlink, "5"},
		{"reverse", tcell.AttrReverse, "7"},
		{"strike", tcell.AttrStrikeThrough, "9"},
	} {
		got := sgr(0, 0, tc.attr, tcell.UnderlineStyleNone)
		if !strings.Contains(got, ";"+tc.param+";") {
			t.Errorf("%s: %q does not carry parameter %s", tc.name, got, tc.param)
		}
	}
}

// A styled underline still has to underline. Only plain 4 is emitted, so the
// style is lost, but the attribute must not be.
func TestSGRUnderlineStylesStillUnderline(t *testing.T) {
	for _, us := range []tcell.UnderlineStyle{
		tcell.UnderlineStyleSolid,
		tcell.UnderlineStyleDouble,
		tcell.UnderlineStyleCurly,
		tcell.UnderlineStyleDotted,
		tcell.UnderlineStyleDashed,
	} {
		if got := sgr(0, 0, tcell.AttrNone, us); !strings.Contains(got, ";4;") {
			t.Errorf("underline style %v produced %q, with no underline parameter", us, got)
		}
	}
	if got := sgr(0, 0, tcell.AttrNone, tcell.UnderlineStyleNone); strings.Contains(got, ";4;") {
		t.Errorf("no underline was asked for, but got %q", got)
	}
}

// No sub-parameter may reach the emulator: a 4:3 that it does not understand
// can swallow the parameters after it, taking the colors with it.
func TestSGRHasNoSubParameters(t *testing.T) {
	for _, us := range []tcell.UnderlineStyle{tcell.UnderlineStyleCurly, tcell.UnderlineStyleDotted} {
		if got := sgr(0xffffff, 0, tcell.AttrUnderline, us); strings.Contains(got, ":") {
			t.Errorf("sgr emitted a sub-parameter: %q", got)
		}
	}
}

// ColorDefault reaches clearScreen as -1, and a literal "-1" in the sequence
// would be read as some other parameter entirely.
func TestNegativeColorsFallBack(t *testing.T) {
	got := clearScreenSeq(-1, -1)
	if strings.Contains(got, "-1") {
		t.Fatalf("negative color leaked into the sequence: %q", got)
	}
	if !strings.Contains(got, "38;2;229;229;229") {
		t.Errorf("default foreground missing from %q", got)
	}
	if !strings.Contains(got, "48;2;0;0;0") {
		t.Errorf("default background missing from %q", got)
	}
}

// A blank cell still has to be painted or the previous contents survive.
func TestEmptyCellIsPainted(t *testing.T) {
	got := drawCellSeq(3, 1, "", 0, 0, tcell.AttrNone, tcell.UnderlineStyleNone)
	if !strings.HasSuffix(got, " ") {
		t.Errorf("empty cell produced %q, which writes nothing", got)
	}
}

func TestDrawCellOrdersPositionThenStyleThenText(t *testing.T) {
	got := drawCellSeq(2, 0, "x", 0xffffff, 0, tcell.AttrBold, tcell.UnderlineStyleNone)
	pos, style := strings.Index(got, "H"), strings.Index(got, "m")
	if pos < 0 || style < 0 || pos > style || !strings.HasSuffix(got, "x") {
		t.Errorf("expected position, then style, then text; got %q", got)
	}
}

func TestShowCursorHidesOnNegative(t *testing.T) {
	if got, want := showCursorSeq(-1, -1), "\x1b[?25l"; got != want {
		t.Errorf("showCursorSeq(-1,-1) = %q, want %q", got, want)
	}
	got := showCursorSeq(4, 2)
	if !strings.HasPrefix(got, "\x1b[?25h") || !strings.HasSuffix(got, cup(4, 2)) {
		t.Errorf("showCursorSeq(4,2) = %q, want show then move", got)
	}
}

// Every class tcell can pass must map to a real DECSCUSR parameter; an
// unknown one must not be silently rendered as some other cursor.
func TestCursorStylesCoverTcell(t *testing.T) {
	for _, class := range []string{
		"cursor-blinking-block", "cursor-steady-block",
		"cursor-blinking-underline", "cursor-steady-underline",
		"cursor-blinking-bar", "cursor-steady-bar",
	} {
		if _, ok := cursorStyles[class]; !ok {
			t.Errorf("no DECSCUSR parameter for %q", class)
		}
	}
	if got, want := setCursorStyleSeq("nonsense"), "\x1b[0 q"; got != want {
		t.Errorf("unknown class = %q, want the terminal default %q", got, want)
	}
}

func TestSetTitle(t *testing.T) {
	if got, want := setTitleSeq("hi"), "\x1b]2;hi\x07"; got != want {
		t.Errorf("setTitleSeq = %q, want %q", got, want)
	}
}

// The append forms exist only to avoid allocating, so they have to produce
// exactly what the string forms do. If they drift, the terminal renders
// differently and nothing else would say so.
func TestAppendFormsMatchStringForms(t *testing.T) {
	styles := []tcell.UnderlineStyle{
		tcell.UnderlineStyleNone, tcell.UnderlineStyleSolid, tcell.UnderlineStyleCurly,
	}
	attrs := []tcell.AttrMask{
		tcell.AttrNone, tcell.AttrBold, tcell.AttrDim, tcell.AttrItalic,
		tcell.AttrBlink, tcell.AttrReverse, tcell.AttrStrikeThrough,
		tcell.AttrUnderline, tcell.AttrBold | tcell.AttrReverse | tcell.AttrDim,
	}
	colors := []int{-1, 0, 0xffffff, 0xff8000, 0x102030}

	for _, a := range attrs {
		for _, us := range styles {
			for _, fg := range colors {
				for _, bg := range colors {
					want := sgr(fg, bg, a, us)
					got := string(appendSGR(nil, fg, bg, a, us))
					if got != want {
						t.Fatalf("appendSGR(%d,%d,%v,%v) = %q, want %q", fg, bg, a, us, got, want)
					}
				}
			}
		}
	}

	for _, p := range [][2]int{{0, 0}, {9, 4}, {79, 23}, {199, 55}} {
		want := cup(p[0], p[1])
		got := string(appendCUP(nil, p[0], p[1]))
		if got != want {
			t.Errorf("appendCUP(%d,%d) = %q, want %q", p[0], p[1], got, want)
		}
	}
}

// The point of the append forms is that a reused buffer stops allocating.
func TestAppendFormsDoNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 4096)
	n := testing.AllocsPerRun(200, func() {
		buf = buf[:0]
		for x := 0; x < 80; x++ {
			buf = appendCUP(buf, x, 3)
			buf = appendSGR(buf, 0xff8000, 0x101010, tcell.AttrBold, tcell.UnderlineStyleNone)
			buf = append(buf, 'x')
		}
	})
	if n != 0 {
		t.Errorf("appending a row of 80 cells allocated %v times, want 0", n)
	}
}
