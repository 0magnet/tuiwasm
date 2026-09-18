//go:build js && wasm

package xtcell

import (
	"syscall/js"

	"github.com/gdamore/tcell/v3"
)

// Mouse reporting. Pointer events on the terminal's element become tcell
// mouse events: button presses and releases (the release is the event
// whose buttons no longer include the one that went down, exactly as
// tcell reports it) and wheel motion. Motion reporting — MouseMotionEvents,
// drag tracking — is not implemented; a TUI that only clicks and scrolls,
// which is most of them, works as it does in a real terminal.
//
// Unlike the keyboard, the pointer is spatial: the listeners live on this
// screen's own element, so there is no claim to arbitrate between screens —
// an event that lands here is ours. There is still one to make against the
// terminal underneath, which would otherwise read the same press as the start
// of a text selection; see claimMouse.

// buttonsMask maps the DOM MouseEvent.buttons bitmask onto tcell's.
func buttonsMask(b int) tcell.ButtonMask {
	var m tcell.ButtonMask
	if b&1 != 0 {
		m |= tcell.ButtonPrimary
	}
	if b&2 != 0 {
		m |= tcell.ButtonSecondary
	}
	if b&4 != 0 {
		m |= tcell.ButtonMiddle
	}
	return m
}

// wheelMask maps wheel deltas onto tcell's wheel "buttons".
func wheelMask(dx, dy float64) tcell.ButtonMask {
	var m tcell.ButtonMask
	switch {
	case dy < 0:
		m |= tcell.WheelUp
	case dy > 0:
		m |= tcell.WheelDown
	}
	switch {
	case dx < 0:
		m |= tcell.WheelLeft
	case dx > 0:
		m |= tcell.WheelRight
	}
	return m
}

func mouseMods(ev js.Value) tcell.ModMask {
	var mod tcell.ModMask
	if ev.Get("shiftKey").Bool() {
		mod |= tcell.ModShift
	}
	if ev.Get("altKey").Bool() {
		mod |= tcell.ModAlt
	}
	if ev.Get("ctrlKey").Bool() {
		mod |= tcell.ModCtrl
	}
	if ev.Get("metaKey").Bool() {
		mod |= tcell.ModMeta
	}
	return mod
}

// gridRect is the box the cells are actually drawn in.
//
// Not the mount element: xterm-go lays the grid out inside it beside a
// scrollbar, so the element is the wider of the two by whatever the scrollbar
// takes. Dividing by the element's width stretches every column slightly and
// the error accumulates rightward along the row — at the right-hand edge of a
// 270-column terminal it names a cell two to the left of the one under the
// pointer, which is enough to miss a link.
func (s *Screen) gridRect() js.Value {
	if screen := s.el.Call("querySelector", ".xterm-screen"); screen.Truthy() {
		return screen.Call("getBoundingClientRect")
	}
	return s.el.Call("getBoundingClientRect")
}

// cellAt turns an event's client coordinates into a cell position. The
// terminal is a uniform grid filling its element, so the division is the
// geometry; the emulator's own font metrics never need to be asked.
func (s *Screen) cellAt(ev js.Value) (int, int) {
	rect := s.gridRect()
	relX := ev.Get("clientX").Float() - rect.Get("left").Float()
	relY := ev.Get("clientY").Float() - rect.Get("top").Float()
	w, h := rect.Get("width").Float(), rect.Get("height").Float()
	s.mu.Lock()
	cols, rows := s.cols, s.rows
	s.mu.Unlock()
	if w <= 0 || h <= 0 || cols <= 0 || rows <= 0 {
		return 0, 0
	}
	x := int(relX * float64(cols) / w)
	y := int(relY * float64(rows) / h)
	if x < 0 {
		x = 0
	}
	if x >= cols {
		x = cols - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= rows {
		y = rows - 1
	}
	return x, y
}

// bindMouse attaches the pointer handlers. They are always attached and
// gated by the EnableMouse flag at event time, so enabling the mouse
// before or after Init both work.
func (s *Screen) bindMouse(el js.Value) {
	if !el.Truthy() {
		// No element — the fake terminal in the tests. Keys have a
		// textarea to find or not; the pointer has only the element.
		return
	}
	button := func(_ js.Value, a []js.Value) any {
		if len(a) == 0 || !s.mouseOn {
			return nil
		}
		ev := a[0]
		x, y := s.cellAt(ev)
		s.post(tcell.NewEventMouse(x, y, buttonsMask(ev.Get("buttons").Int()), mouseMods(ev)))
		return nil
	}
	s.mdown = js.FuncOf(button)
	s.mup = js.FuncOf(button)
	s.mwheel = js.FuncOf(func(_ js.Value, a []js.Value) any {
		if len(a) == 0 || !s.mouseOn {
			return nil
		}
		ev := a[0]
		wheel := wheelMask(ev.Get("deltaX").Float(), ev.Get("deltaY").Float())
		if wheel == 0 {
			return nil
		}
		x, y := s.cellAt(ev)
		s.post(tcell.NewEventMouse(x, y, wheel|buttonsMask(ev.Get("buttons").Int()), mouseMods(ev)))
		// The page must not scroll under the terminal.
		ev.Call("preventDefault")
		return nil
	})
	el.Call("addEventListener", "mousedown", s.mdown, true)
	el.Call("addEventListener", "mouseup", s.mup, true)
	// passive:false, or preventDefault is ignored for wheel events.
	el.Call("addEventListener", "wheel", s.mwheel, map[string]any{"passive": false, "capture": true})
}

func (s *Screen) detachMouse() {
	if !s.el.Truthy() {
		return
	}
	for _, h := range []struct {
		name string
		fn   *js.Func
	}{{"mousedown", &s.mdown}, {"mouseup", &s.mup}, {"wheel", &s.mwheel}} {
		if h.fn.Truthy() {
			s.el.Call("removeEventListener", h.name, *h.fn, true)
			h.fn.Release()
			*h.fn = js.Func{}
		}
	}
}

// EnableMouse turns on mouse reporting: button presses, releases and the
// wheel. Motion reporting is not implemented, so MouseMotionEvents (and
// drag tracking) are quietly less than a real terminal offers.
func (s *Screen) EnableMouse(...tcell.MouseFlags) {
	s.mouseOn = true
	s.claimMouse(true)
}

// DisableMouse stops mouse reporting.
func (s *Screen) DisableMouse() {
	s.mouseOn = false
	s.claimMouse(false)
}

// mouseClaimer is a terminal that can be told an application has taken the
// pointer. Optional, because the fake terminal in the tests is not one.
type mouseClaimer interface{ ClaimMouse(bool) }

// claimMouse tells the terminal that the pointer belongs to the application
// while this screen has the mouse enabled.
//
// The listeners above run in the capture phase, so the tcell event is posted
// either way — but the terminal's own handlers still run afterwards on the
// same click, and an emulator with no application asking for the mouse treats
// a press as the start of a text selection. The result was that every click on
// a link also smeared a selection across the screen, and it stayed there.
//
// Saying so through the core mouse service rather than swallowing the event is
// what makes shift-to-select keep working: xterm-go already yields the pointer
// to an application that has asked for it and takes it back for a shifted
// click, which is the convention every terminal emulator uses for exactly this
// conflict. It also stops the wheel scrolling the emulator's scrollback out
// from under a full-screen TUI.
//
// The reports the terminal now encodes go nowhere: bindInput has already
// replaced OnData for as long as this screen is running.
func (s *Screen) claimMouse(on bool) {
	if c, ok := s.term.(mouseClaimer); ok {
		c.ClaimMouse(on)
	}
}
