// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/exp/shiny/driver/internal/pump"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
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

	pump    pump.Pump
	xevents chan xgb.Event

	// This next group of variables are mutable, but are only modified in the
	// screenImpl.run goroutine.
	width, height int

	mu       sync.Mutex
	released bool
}

func (w *windowImpl) Events() <-chan interface{} { return w.pump.Events() }
func (w *windowImpl) Send(event interface{})     { w.pump.Send(event) }

func (w *windowImpl) Release() {
	w.mu.Lock()
	released := w.released
	w.released = true
	w.mu.Unlock()

	if released {
		return
	}
	render.FreePicture(w.s.xc, w.xp)
	xproto.FreeGC(w.s.xc, w.xg)
	xproto.DestroyWindow(w.s.xc, w.xw)
	w.pump.Release()
}

func (w *windowImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle, sender screen.Sender) {
	src.(*bufferImpl).upload(w, xproto.Drawable(w.xw), w.xg, w.s.xsi.RootDepth, dp, sr, sender)
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	fill(w.s.xc, w.xp, dr, src, op)
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	src.(*textureImpl).draw(w.xp, &src2dst, sr, op, opts)
}

func (w *windowImpl) EndPaint(e paint.Event) {
	// TODO.
}

func (w *windowImpl) handleConfigureNotify(ev xproto.ConfigureNotifyEvent) {
	// TODO: lifecycle events.

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

	// TODO: translate X11 expose events to shiny paint events, instead of
	// sending this synthetic paint event as a hack.
	w.Send(paint.Event{})
}

func (w *windowImpl) handleMouse(x, y int16, b xproto.Button, state uint16, dir mouse.Direction) {
	// TODO: should a mouse.Event have a separate MouseModifiers field, for
	// which buttons are pressed during a mouse move?
	w.Send(mouse.Event{
		X:         float32(x),
		Y:         float32(y),
		Button:    mouse.Button(b),
		Modifiers: keyModifiers(state),
		Direction: dir,
	})
}

// These constants come from /usr/include/X11/X.h
const (
	xShiftMask   = 1 << 0
	xLockMask    = 1 << 1
	xControlMask = 1 << 2
	xMod1Mask    = 1 << 3
	xMod2Mask    = 1 << 4
	xMod3Mask    = 1 << 5
	xMod4Mask    = 1 << 6
	xMod5Mask    = 1 << 7
	xButton1Mask = 1 << 8
	xButton2Mask = 1 << 9
	xButton3Mask = 1 << 10
	xButton4Mask = 1 << 11
	xButton5Mask = 1 << 12
)

func keyModifiers(state uint16) (m key.Modifiers) {
	if state&xShiftMask != 0 {
		m |= key.ModShift
	}
	if state&xControlMask != 0 {
		m |= key.ModControl
	}
	if state&xMod1Mask != 0 {
		m |= key.ModAlt
	}
	if state&xMod4Mask != 0 {
		m |= key.ModMeta
	}
	return m
}
