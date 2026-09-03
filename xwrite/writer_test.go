package xwrite

import "testing"

// Line endings are the whole job, and getting them wrong is the classic
// symptom of Go output in a terminal: every line starts where the last one
// ended, so the text staircases off to the right.
func TestBareNewlineGetsCarriageReturn(t *testing.T) {
	if got, want := string(crlf([]byte("a\nb"))), "a\r\nb"; got != want {
		t.Errorf("crlf = %q, want %q", got, want)
	}
}

func TestExistingCRLFIsNotDoubled(t *testing.T) {
	if got, want := string(crlf([]byte("a\r\nb"))), "a\r\nb"; got != want {
		t.Errorf("crlf = %q, want %q", got, want)
	}
}

// Progress bars redraw by returning to the start of the line without ending
// it. Turning that into a newline would print one line per update instead.
func TestLoneCarriageReturnSurvives(t *testing.T) {
	if got, want := string(crlf([]byte("50%\r75%\r"))), "50%\r75%\r"; got != want {
		t.Errorf("crlf = %q, want %q", got, want)
	}
}

func TestNoNewlinesIsUntouched(t *testing.T) {
	in := []byte("\x1b[1mbold\x1b[0m")
	out := crlf(in)
	if string(out) != string(in) {
		t.Errorf("crlf = %q, want %q", out, in)
	}
	if &out[0] != &in[0] {
		t.Error("input without newlines should be passed through, not copied")
	}
}

func TestLeadingNewline(t *testing.T) {
	if got, want := string(crlf([]byte("\nx"))), "\r\nx"; got != want {
		t.Errorf("crlf = %q, want %q", got, want)
	}
}

func TestConsecutiveNewlines(t *testing.T) {
	if got, want := string(crlf([]byte("a\n\n\nb"))), "a\r\n\r\n\r\nb"; got != want {
		t.Errorf("crlf = %q, want %q", got, want)
	}
}
