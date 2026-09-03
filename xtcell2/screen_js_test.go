//go:build js && wasm

package xtcell2

import (
	"strings"
	"syscall/js"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// fakeTerm is a terminal that records what was written to it instead of
// drawing. It is what lets these tests run without a DOM.
type fakeTerm struct {
	written  strings.Builder
	cols     int
	rows     int
	onResize func(int, int)
	onData   func(string)
}

func (f *fakeTerm) Write(data []byte)            { f.written.Write(data) }
func (f *fakeTerm) Cols() int                    { return f.cols }
func (f *fakeTerm) Rows() int                    { return f.rows }
func (f *fakeTerm) OnResize() func(int, int)     { return f.onResize }
func (f *fakeTerm) SetOnResize(g func(int, int)) { f.onResize = g }
func (f *fakeTerm) OnData() func(string)         { return f.onData }
func (f *fakeTerm) SetOnData(g func(string))     { f.onData = g }

// newTestScreen builds an initialised screen over a fake terminal.
// js.Undefined() as the element makes every DOM call in bindInput a no-op.
func newTestScreen(t *testing.T, cols, rows int) (*Screen, *fakeTerm) {
	t.Helper()
	current = nil
	t.Cleanup(func() { current = nil })

	ft := &fakeTerm{cols: cols, rows: rows}
	s := newScreen(ft, js.Undefined())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	// Init clears the terminal; the tests below are about what comes after.
	ft.written.Reset()
	return s, ft
}

// This is the whole reason the package stopped using tcell's own web screen.
// That one paints by calling a JavaScript global once per cell, which measured
// 79us a cell here — a full redraw of a large window is two orders of magnitude
// past a frame. Nothing may cross the boundary per cell; a frame is one write.
func TestAFrameIsASingleWrite(t *testing.T) {
	s, ft := newTestScreen(t, 40, 10)

	writes := 0
	counting := &countingTerm{fakeTerm: ft, n: &writes}
	s.term = counting

	for y := 0; y < 10; y++ {
		for x := 0; x < 40; x++ {
			s.SetContent(x, y, 'x', nil, tcell.StyleDefault)
		}
	}
	s.Show()

	if writes != 1 {
		t.Errorf("a 400-cell frame took %d writes to the terminal, want 1", writes)
	}
}

type countingTerm struct {
	*fakeTerm
	n *int
}

func (c *countingTerm) Write(b []byte) { *c.n++; c.fakeTerm.Write(b) }

// Only what changed is drawn. Redrawing everything every frame would work, but
// it is the difference between a frame costing a kilobyte and costing a hundred.
func TestOnlyDirtyCellsAreDrawn(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)

	s.SetContent(3, 1, 'a', nil, tcell.StyleDefault)
	s.Show()
	first := ft.written.String()
	if !strings.Contains(first, cup(3, 1)) {
		t.Fatalf("the cell that changed was not drawn: %q", first)
	}

	// Nothing has changed since.
	ft.written.Reset()
	s.Show()
	if got := ft.written.String(); got != "" {
		t.Errorf("an unchanged screen still wrote %q", got)
	}

	// Setting a cell to what it already holds is not a change either.
	s.SetContent(3, 1, 'a', nil, tcell.StyleDefault)
	s.Show()
	if got := ft.written.String(); got != "" {
		t.Errorf("re-setting a cell to its own contents wrote %q", got)
	}

	// One new cell is one cell's worth of output.
	s.SetContent(4, 1, 'b', nil, tcell.StyleDefault)
	s.Show()
	got := ft.written.String()
	if !strings.Contains(got, cup(4, 1)) {
		t.Errorf("the newly changed cell was not drawn: %q", got)
	}
	if strings.Contains(got, cup(3, 1)) {
		t.Errorf("an unchanged cell was redrawn: %q", got)
	}
}

// Consecutive cells do not each say where they are. The cursor is already
// there, having been advanced by the glyph before — and a position is about
// nine bytes against a cell's forty, so in a full-screen animation, where every
// cell is dirty and they are all consecutive, this is a quarter of the frame.
func TestConsecutiveCellsDoNotRepeatThePosition(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)
	for x := 0; x < 5; x++ {
		s.SetContent(x, 0, 'x', nil, tcell.StyleDefault)
	}
	s.Show()

	// One to start the run. The trailing cursor placement is a move of its own.
	if n := strings.Count(ft.written.String(), "\x1b["+"1;1H"); n != 1 {
		t.Errorf("the run began with %d position sequences, want 1", n)
	}
	for x := 1; x < 5; x++ {
		if strings.Contains(ft.written.String(), cup(x, 0)) {
			t.Errorf("cell %d restated its position although it follows the last one", x)
		}
	}
}

// A gap means the cursor is not where the next cell is, so the position has to
// be stated. Getting this wrong writes the frame in the wrong place.
func TestACellAfterAGapStatesItsPosition(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)
	s.SetContent(0, 0, 'a', nil, tcell.StyleDefault)
	s.SetContent(5, 0, 'b', nil, tcell.StyleDefault)
	s.SetContent(0, 2, 'c', nil, tcell.StyleDefault)
	s.Show()

	got := ft.written.String()
	for _, want := range []struct {
		x, y int
	}{{0, 0}, {5, 0}, {0, 2}} {
		if !strings.Contains(got, cup(want.x, want.y)) {
			t.Errorf("the cell at %d,%d did not state its position: %q", want.x, want.y, got)
		}
	}
}

// Sync exists for when the terminal is not showing what this screen last put
// there — after a resize, or after another window wrote over it.
//
// It has to clear first. Invalidate works by forgetting what was last drawn,
// but an empty cell was last drawn as the empty string and still is one, so it
// never comes back dirty; without the clear, everything the screen believes is
// blank keeps whatever the previous occupant left there.
func TestSyncClearsSoBlankCellsComeBack(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)
	s.SetContent(0, 0, 'a', nil, tcell.StyleDefault)
	s.Show()

	ft.written.Reset()
	s.Sync()

	got := ft.written.String()
	if !strings.Contains(got, "\x1b[2J") {
		t.Errorf("Sync did not clear the terminal: %q", got)
	}
	if !strings.Contains(got, cup(0, 0)) {
		t.Errorf("Sync did not repaint the cell that has content: %q", got)
	}
}

// A run of cells sharing a style should say it once. That is most of the bytes
// in a frame, and neighbouring cells nearly always share a style.
func TestRepeatedStyleIsWrittenOnce(t *testing.T) {
	s, ft := newTestScreen(t, 80, 24)
	st := tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(1, 2, 3)).
		Background(tcell.NewRGBColor(4, 5, 6))

	for x := 0; x < 10; x++ {
		s.SetContent(x, 0, 'x', nil, st)
	}
	s.Show()

	want := sgr(0x010203, 0x040506, 0, 0)
	if n := strings.Count(ft.written.String(), want); n != 1 {
		t.Errorf("ten cells of one style wrote the style %d times, want once", n)
	}
}

func TestAChangeOfStyleIsWritten(t *testing.T) {
	s, ft := newTestScreen(t, 80, 24)
	a := tcell.StyleDefault.Foreground(tcell.NewRGBColor(1, 2, 3))
	b := tcell.StyleDefault.Foreground(tcell.NewRGBColor(9, 9, 9))

	s.SetContent(0, 0, 'a', nil, a)
	s.SetContent(1, 0, 'b', nil, b)
	s.Show()

	got := ft.written.String()
	if !strings.Contains(got, "1;2;3") || !strings.Contains(got, "9;9;9") {
		t.Errorf("a change of style was not written: %q", got)
	}
}

// The size is the terminal's, and it stays the terminal's. tcell's own web
// screen hardcodes 80x24 and offers no way to say otherwise, which is what the
// previous adapter had to work around by pushing a size in from outside.
func TestSizeComesFromTheTerminal(t *testing.T) {
	s, _ := newTestScreen(t, 132, 43)
	if c, r := s.Size(); c != 132 || r != 43 {
		t.Errorf("screen is %dx%d, the terminal is 132x43", c, r)
	}
}

func TestResizeFollowsTheTerminalAndPostsAnEvent(t *testing.T) {
	s, ft := newTestScreen(t, 80, 24)
	if ft.onResize == nil {
		t.Fatal("Init did not subscribe to resizes")
	}

	ft.onResize(100, 30)

	if c, r := s.Size(); c != 100 || r != 30 {
		t.Errorf("after a resize the screen is %dx%d, want 100x30", c, r)
	}
	ev := s.PollEvent()
	re, ok := ev.(*tcell.EventResize)
	if !ok {
		t.Fatalf("a resize produced %T, want *tcell.EventResize", ev)
	}
	if c, r := re.Size(); c != 100 || r != 30 {
		t.Errorf("the resize event says %dx%d, want 100x30", c, r)
	}
}

// Init chains onto whatever the terminal already had rather than replacing it,
// so the terminal's own resize handling keeps working.
func TestResizeDoesNotDropAnExistingHandler(t *testing.T) {
	current = nil
	t.Cleanup(func() { current = nil })

	ft := &fakeTerm{cols: 80, rows: 24}
	called := 0
	ft.SetOnResize(func(int, int) { called++ })

	s := newScreen(ft, js.Undefined())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	ft.onResize(90, 25)

	if called != 1 {
		t.Errorf("the handler that was already there was called %d times, want once", called)
	}
}

// A zero size means the terminal has not been laid out yet. Taking it would
// tell the program it has no room at all.
func TestAZeroSizedTerminalGetsAUsableDefault(t *testing.T) {
	current = nil
	t.Cleanup(func() { current = nil })

	s := newScreen(&fakeTerm{cols: 0, rows: 0}, js.Undefined())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if c, r := s.Size(); c <= 0 || r <= 0 {
		t.Errorf("a terminal that has not been measured gave the program %dx%d", c, r)
	}
}

// The limitation this design removes. tcell's web screen keeps its painting
// functions on the JavaScript global object, so a page had room for exactly one
// of them and a second window opening left the first painting into the wrong
// terminal. These screens share nothing.
func TestTwoScreensDrawToTheirOwnTerminals(t *testing.T) {
	a, ta := newTestScreen(t, 20, 5)
	b, tb := newTestScreen(t, 20, 5)

	a.SetContent(0, 0, 'A', nil, tcell.StyleDefault)
	b.SetContent(0, 0, 'B', nil, tcell.StyleDefault)
	a.Show()
	b.Show()

	if got := ta.written.String(); !strings.Contains(got, "A") || strings.Contains(got, "B") {
		t.Errorf("the first terminal received %q, want only its own cell", got)
	}
	if got := tb.written.String(); !strings.Contains(got, "B") || strings.Contains(got, "A") {
		t.Errorf("the second terminal received %q, want only its own cell", got)
	}
}

// Closing ch is the contract a caller relies on to notice a screen going away:
// an animation loop selects on it and returns when it closes, which is how a
// closed window ends its goroutine instead of leaving it spinning.
func TestChannelEventsClosesOnFini(t *testing.T) {
	s, _ := newTestScreen(t, 20, 5)

	ch := make(chan tcell.Event)
	go s.ChannelEvents(ch, make(chan struct{}))
	s.Fini()

	for range ch { //nolint:revive // draining until closed is the point
	}
}

// Fini gives the terminal back: xterm-go's own data callback was taken away
// while the screen was running so keystrokes did not arrive twice.
func TestFiniRestoresTheTerminalsDataCallback(t *testing.T) {
	current = nil
	t.Cleanup(func() { current = nil })

	ft := &fakeTerm{cols: 20, rows: 5}
	marker := func(string) {}
	ft.SetOnData(marker)

	s := newScreen(ft, js.Undefined())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if ft.onData == nil {
		t.Fatal("Init did not take the data callback")
	}
	s.Fini()
	if ft.onData == nil {
		t.Error("Fini did not give the data callback back")
	}
}

// Fini must be safe to call more than once: a window closing and a demo ending
// can both reach it.
func TestFiniIsIdempotent(t *testing.T) {
	s, _ := newTestScreen(t, 20, 5)
	s.Fini()
	s.Fini()
}

// A control chord has to arrive as the control key, because that is what
// programs check for: an animation quits on tcell.KeyCtrlC, not on a rune.
//
// decodeKey itself returns the plain rune with ModCtrl and lets NewEventKey do
// the conversion, so the assertion belongs on the event rather than on the
// decode — that is the value the program actually sees.
func TestControlChordsArriveAsControlKeys(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want tcell.Key
	}{
		{"c", tcell.KeyCtrlC},
		{"C", tcell.KeyCtrlC},
		{"a", tcell.KeyCtrlA},
		{"z", tcell.KeyCtrlZ},
	} {
		k, r, mod, ok := decodeKey(tc.key, false, false, true, false)
		if !ok {
			t.Errorf("ctrl-%s produced no event", tc.key)
			continue
		}
		ev := tcell.NewEventKey(k, r, mod)
		if ev.Key() != tc.want {
			t.Errorf("ctrl-%s arrives as key %v, want %v", tc.key, ev.Key(), tc.want)
		}
	}
}

func TestNamedKeysUseTcellsOwnTable(t *testing.T) {
	k, _, _, ok := decodeKey("ArrowUp", false, false, false, false)
	if !ok || k != tcell.KeyUp {
		t.Errorf("ArrowUp decoded to %v, want KeyUp", k)
	}
}

func TestPlainRunesArriveAsRunes(t *testing.T) {
	k, r, mod, ok := decodeKey("q", false, false, false, false)
	if !ok || k != tcell.KeyRune || r != 'q' || mod != tcell.ModNone {
		t.Errorf("q decoded to key %v rune %q mod %v", k, r, mod)
	}
}

// A modifier held on its own is not a keystroke; delivering it would make every
// shifted letter arrive as two events.
func TestModifiersAloneAreNotEvents(t *testing.T) {
	for _, k := range []string{"Control", "Alt", "Meta", "Shift"} {
		if _, _, _, ok := decodeKey(k, false, false, false, false); ok {
			t.Errorf("%s on its own produced an event", k)
		}
	}
}

// An empty cell is a space. Writing nothing would leave whatever was there.
func TestAnEmptyCellIsASpace(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)
	s.cell(0, 0, "", tcell.StyleDefault)
	s.mu.Lock()
	s.flushLocked()
	s.mu.Unlock()
	if !strings.HasSuffix(ft.written.String(), " ") {
		t.Errorf("an empty cell did not write a space: %q", ft.written.String())
	}
}

// The buffer keeps its capacity between frames — that is what it is for — but
// must not carry a frame's content into the next one.
func TestFlushEmptiesTheBuffer(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)

	s.mu.Lock()
	s.buf = append(s.buf, "first"...)
	s.flushLocked()
	s.buf = append(s.buf, "second"...)
	s.flushLocked()
	s.mu.Unlock()

	if got := ft.written.String(); got != "firstsecond" {
		t.Errorf("terminal received %q, want the two frames once each", got)
	}
}

func TestFlushOnAnEmptyBufferWritesNothing(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)
	s.mu.Lock()
	s.flushLocked()
	s.flushLocked()
	s.mu.Unlock()
	if ft.written.Len() != 0 {
		t.Errorf("flushing an empty buffer wrote %q", ft.written.String())
	}
}

// Suspend on tcell v2.13.10's own web screen takes the screen's lock and never
// releases it, so every later Show or key event on that screen blocks forever —
// and in a page, where Go has one thread, that stops the whole tab. This one
// has to survive being suspended and resumed.
func TestSuspendAndResumeLeaveTheScreenUsable(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)

	if err := s.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := s.Resume(); err != nil {
		t.Fatal(err)
	}

	// If the lock had leaked, this would never return.
	ft.written.Reset()
	s.SetContent(0, 0, 'z', nil, tcell.StyleDefault)
	s.Show()
	if !strings.Contains(ft.written.String(), "z") {
		t.Error("the screen did not draw after being suspended and resumed")
	}
}

// HideCursor is what an animation uses to get the blinking block off its
// picture, so it has to reach the terminal.
func TestHideCursorReachesTheTerminal(t *testing.T) {
	s, ft := newTestScreen(t, 20, 5)
	s.HideCursor()
	s.SetContent(0, 0, 'a', nil, tcell.StyleDefault)
	s.Show()
	if !strings.Contains(ft.written.String(), "\x1b[?25l") {
		t.Errorf("the cursor was not hidden: %q", ft.written.String())
	}
}

// sinkTerm is a terminal that takes cells directly, which is what the real one
// does. It records them instead of drawing.
type sinkTerm struct {
	*fakeTerm
	cells    map[[2]int]string
	refreshs [][2]int
}

func newSinkTerm(cols, rows int) *sinkTerm {
	return &sinkTerm{
		fakeTerm: &fakeTerm{cols: cols, rows: rows},
		cells:    map[[2]int]string{},
	}
}

func (k *sinkTerm) putCell(x, y int, str string, fg, bg uint32, width int) {
	k.cells[[2]int{x, y}] = str
}
func (k *sinkTerm) refresh(y1, y2 int) { k.refreshs = append(k.refreshs, [2]int{y1, y2}) }

func newDirectScreen(t *testing.T, cols, rows int) (*Screen, *sinkTerm) {
	t.Helper()
	current = nil
	t.Cleanup(func() { current = nil })

	kt := newSinkTerm(cols, rows)
	s := newScreen(kt, js.Undefined())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if s.sink == nil {
		t.Fatal("a terminal that takes cells directly was not recognised")
	}
	return s, kt
}

// The direct path is the one that made the pixel animations viable: the
// emulator is in this process, so its cells are written where they were going
// to end up rather than encoded as escape sequences and parsed straight back.
func TestDirectPathWritesCellsAndRefreshesTheRowsItTouched(t *testing.T) {
	s, kt := newDirectScreen(t, 20, 5)

	s.SetContent(2, 1, 'a', nil, tcell.StyleDefault)
	s.SetContent(7, 3, 'b', nil, tcell.StyleDefault)
	s.Show()

	if got := kt.cells[[2]int{2, 1}]; got != "a" {
		t.Errorf("cell 2,1 is %q, want %q", got, "a")
	}
	if got := kt.cells[[2]int{7, 3}]; got != "b" {
		t.Errorf("cell 7,3 is %q, want %q", got, "b")
	}
	if len(kt.cells) != 2 {
		t.Errorf("%d cells were written, want only the 2 that changed", len(kt.cells))
	}
	if len(kt.refreshs) != 1 || kt.refreshs[0] != [2]int{1, 3} {
		t.Errorf("rows refreshed: %v, want one span covering rows 1 to 3", kt.refreshs)
	}
}

// Nothing changed means nothing is written and the renderer is not woken.
func TestDirectPathDoesNothingWhenNothingChanged(t *testing.T) {
	s, kt := newDirectScreen(t, 20, 5)
	s.SetContent(0, 0, 'a', nil, tcell.StyleDefault)
	s.Show()

	kt.cells = map[[2]int]string{}
	kt.refreshs = nil
	s.Show()

	if len(kt.cells) != 0 || len(kt.refreshs) != 0 {
		t.Errorf("an unchanged screen wrote %d cells and %d refreshes", len(kt.cells), len(kt.refreshs))
	}
}

// The cursor has to be stated rather than assumed.
//
// tcell starts with the cursor hidden and so does this screen, so HideCursor is
// a no-op — but the terminal starts with its cursor showing. A screen that only
// believes the cursor is hidden and never says so leaves a block blinking in
// the corner of every animation, which is exactly what it did: on the direct
// path nothing else ever moves the cursor, so there was no second chance to
// correct it.
func TestTheCursorStateIsStatedOnTheFirstFrame(t *testing.T) {
	s, kt := newDirectScreen(t, 20, 5)
	s.HideCursor() // what every animation does, and what changes nothing
	s.SetContent(0, 0, 'a', nil, tcell.StyleDefault)
	s.Show()

	if !strings.Contains(kt.written.String(), "\x1b[?25l") {
		t.Errorf("the first frame did not hide the cursor: %q", kt.written.String())
	}
}

// Having said it once, it must not keep saying it.
func TestTheCursorIsNotRestatedEveryFrame(t *testing.T) {
	s, kt := newDirectScreen(t, 20, 5)
	s.SetContent(0, 0, 'a', nil, tcell.StyleDefault)
	s.Show()

	kt.written.Reset()
	for i := 0; i < 5; i++ {
		s.SetContent(i, 1, 'x', nil, tcell.StyleDefault)
		s.Show()
	}
	if n := strings.Count(kt.written.String(), "\x1b[?25l"); n != 0 {
		t.Errorf("the cursor state was restated %d times after nothing changed about it", n)
	}
}

// A Sync queues a clear-screen. On the direct path the cells do not go through
// that buffer, so if the clear were flushed afterwards it would erase the very
// frame it was meant to precede.
func TestSyncClearsBeforeTheCellsAreWritten(t *testing.T) {
	s, kt := newDirectScreen(t, 20, 5)
	s.SetContent(0, 0, 'a', nil, tcell.StyleDefault)
	s.Show()

	kt.written.Reset()
	kt.cells = map[[2]int]string{}
	s.Sync()

	if !strings.Contains(kt.written.String(), "\x1b[2J") {
		t.Errorf("Sync did not clear: %q", kt.written.String())
	}
	if got := kt.cells[[2]int{0, 0}]; got != "a" {
		t.Errorf("after Sync the cell holds %q, want it repainted as %q", got, "a")
	}
}
