// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

// TODO: implement a back buffer.

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/exp/shiny/driver/internal/drawer"
	"golang.org/x/exp/shiny/driver/internal/event"
	"golang.org/x/exp/shiny/driver/internal/x11key"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/geom"
)

type windowImpl struct {
	s *screenImpl

	xw xproto.Window
	xg xproto.Gcontext
	xp render.Picture

	event.Queue
	xevents chan xgb.Event

	// This next group of variables are mutable, but are only modified in the
	// screenImpl.run goroutine.
	width, height int

	mu       sync.Mutex
	stage    lifecycle.Stage
	dead     bool
	visible  bool
	focused  bool
	released bool
}

func (w *windowImpl) sendLifecycle() {
	w.mu.Lock()
	from, to := w.stage, lifecycle.StageAlive
	// The order of these if's is important. For example, once a window becomes
	// StageDead, it should never change stage again.
	//
	// Similarly, focused trumps visible. It's hard to imagine a situation
	// where an X11 window is focused and not visible on screen, but in that
	// unlikely case, StageFocused seems the most appropriate stage.
	if w.dead {
		to = lifecycle.StageDead
	} else if w.focused {
		to = lifecycle.StageFocused
	} else if w.visible {
		to = lifecycle.StageVisible
	}
	w.stage = to
	w.mu.Unlock()

	if from != to {
		w.Send(lifecycle.Event{
			From: from,
			To:   to,
		})
	}
}

func (w *windowImpl) Release() {
	w.mu.Lock()
	released := w.released
	w.released = true
	w.mu.Unlock()

	// TODO: set w.dead and call w.sendLifecycle, a la handling atomWMDeleteWindow?

	if released {
		return
	}
	render.FreePicture(w.s.xc, w.xp)
	xproto.FreeGC(w.s.xc, w.xg)
	xproto.DestroyWindow(w.s.xc, w.xw)
}

func (w *windowImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle) {
	src.(*bufferImpl).upload(w, xproto.Drawable(w.xw), w.xg, w.s.xsi.RootDepth, dp, sr)
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	fill(w.s.xc, w.xp, dr, src, op)
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	src.(*textureImpl).draw(w.xp, &src2dst, sr, op, w.width, w.height, opts)
}

func (w *windowImpl) Copy(dp image.Point, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	drawer.Copy(w, dp, src, sr, op, opts)
}

func (w *windowImpl) Scale(dr image.Rectangle, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	drawer.Scale(w, dr, src, sr, op, opts)
}

func (w *windowImpl) Publish() screen.PublishResult {
	// TODO.
	return screen.PublishResult{}
}

func (w *windowImpl) handleConfigureNotify(ev xproto.ConfigureNotifyEvent) {
	// TODO: does the order of these lifecycle and size events matter? Should
	// they really be a single, atomic event?
	w.mu.Lock()
	w.visible = (int(ev.X)+int(ev.Width)) >= 0 && (int(ev.Y)+int(ev.Height)) >= 0
	w.mu.Unlock()

	w.sendLifecycle()

	newWidth, newHeight := int(ev.Width), int(ev.Height)
	if w.width == newWidth && w.height == newHeight {
		return
	}
	w.width, w.height = newWidth, newHeight
	// TODO: don't assume that PixelsPerPt == 1.
	w.Send(size.Event{
		WidthPx:     newWidth,
		HeightPx:    newHeight,
		WidthPt:     geom.Pt(newWidth),
		HeightPt:    geom.Pt(newHeight),
		PixelsPerPt: 1,
	})
}

func (w *windowImpl) handleExpose() {
	w.Send(paint.Event{})
}

func (w *windowImpl) handleKey(detail xproto.Keycode, state uint16, dir key.Direction) {
	// The key event's rune depends on whether the shift key is down.
	unshifted := rune(w.s.keysyms[detail][0])
	r := unshifted
	if state&x11key.ShiftMask != 0 {
		r = rune(w.s.keysyms[detail][1])
		// In X11, a zero xproto.Keysym when shift is down means to use what
		// the xproto.Keysym is when shift is up.
		if r == 0 {
			r = unshifted
		}
	}

	// The key event's code is independent of whether the shift key is down.
	var c key.Code
	if 0 <= unshifted && unshifted < 0x80 {
		// TODO: distinguish the regular '2' key and number-pad '2' key (with
		// Num-Lock).
		c = x11key.ASCIIKeycodes[unshifted]
	} else {
		r, c = -1, x11key.NonUnicodeKeycodes[unshifted]
	}

	// TODO: Unicode-but-not-ASCII keysyms like the Swiss keyboard's 'ö'.

	w.Send(key.Event{
		Rune:      r,
		Code:      c,
		Modifiers: x11key.KeyModifiers(state),
		Direction: dir,
	})
}

func (w *windowImpl) handleMouse(x, y int16, b xproto.Button, state uint16, dir mouse.Direction) {
	// TODO: should a mouse.Event have a separate MouseModifiers field, for
	// which buttons are pressed during a mouse move?
	w.Send(mouse.Event{
		X:         float32(x),
		Y:         float32(y),
		Button:    mouse.Button(b),
		Modifiers: x11key.KeyModifiers(state),
		Direction: dir,
	})
}
