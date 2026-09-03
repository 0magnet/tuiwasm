package anim

import "testing"

// Only the katakana are mirrored. The film's numerals are flipped too, but
// Unicode cannot express that either and flipping a digit in a terminal would
// just make it unreadable — so the set is drawn with its kana turned round and
// nothing else.
func TestOnlyKatakanaAreMirrored(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
		why  string
	}{
		{"ｱ", true, "halfwidth katakana A"},
		{"ﾝ", true, "halfwidth katakana N"},
		{"ｦ", true, "the first of the range"},
		{"ﾟ", true, "the last of the range"},
		{"0", false, "a digit"},
		{"9", false, "a digit"},
		{"Z", false, "the one Latin letter in the set"},
		{"ア", false, "FULL-width katakana is a different character"},
		{"｡", false, "halfwidth punctuation below the kana"},
		{"│", false, "box drawing, which other demos rely on"},
		{" ", false, "a space"},
		{"", false, "nothing"},
	} {
		if got := mirrorKana(tc.in); got != tc.want {
			t.Errorf("mirrorKana(%q) = %v, want %v — %s", tc.in, got, tc.want, tc.why)
		}
	}
}

// Every other demo must leave the terminal alone: a mirrored box-drawing
// character or a mirrored Z would be a bug in someone else's demo.
func TestOnlyTheMatrixDemoAsksForMirroring(t *testing.T) {
	// The registry is populated by this package's init.
	var asked []string
	for _, d := range demoList() {
		if d.Mirror != nil {
			asked = append(asked, d.Name)
		}
	}
	if len(asked) != 1 || asked[0] != "matrix" {
		t.Errorf("demos asking to be mirrored: %v, want just [matrix]", asked)
	}
}
