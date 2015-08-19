// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

// #include "windriver.h"
import "C"

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"golang.org/x/exp/shiny/driver/internal/pump"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/geom"
)

var (
	windows     = map[C.HWND]*window{}
	windowsLock sync.Mutex
)

type window struct {
	hwnd C.HWND
	pump pump.Pump
}

func newWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	var hwnd C.HWND

	hr := C.createWindow(&hwnd)
	if hr != C.S_OK {
		return nil, winerror("error creating window", hr)
	}

	w := &window{
		hwnd: hwnd,
		pump: pump.Make(),
	}

	windowsLock.Lock()
	windows[hwnd] = w
	windowsLock.Unlock()

	return w, nil
}

func (w *window) Release() {
	if w.hwnd == nil { // already released?
		return
	}

	windowsLock.Lock()
	delete(windows, w.hwnd)
	windowsLock.Unlock()

	// TODO(andlabs): check for errors from this?
	C.destroyWindow(w.hwnd)
	w.hwnd = nil
	w.pump.Release()
}

func (w *window) Events() <-chan interface{} { return w.pump.Events() }
func (w *window) Send(event interface{})     { w.pump.Send(event) }

func (w *window) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle, sender screen.Sender) {
	// TODO
}

func (w *window) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	// TODO
}

func (w *window) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	// TODO
}

func (w *window) EndPaint(p paint.Event) {
	// TODO
}

//export sendSizeEvent
func sendSizeEvent(hwnd C.HWND, r *C.RECT) {
	windowsLock.Lock()
	w := windows[hwnd]
	windowsLock.Unlock()

	width := int(r.right - r.left)
	height := int(r.bottom - r.top)
	// TODO(andlabs): don't assume that PixelsPerPt == 1
	w.Send(size.Event{
		WidthPx:     width,
		HeightPx:    height,
		WidthPt:     geom.Pt(width),
		HeightPt:    geom.Pt(height),
		PixelsPerPt: 1,
	})
}

//export sendMouseEvent
func sendMouseEvent(hwnd C.HWND, uMsg C.UINT, x C.int, y C.int) {
	var dir mouse.Direction
	var button mouse.Button

	windowsLock.Lock()
	w := windows[hwnd]
	windowsLock.Unlock()

	switch uMsg {
	case C.WM_MOUSEMOVE:
		dir = mouse.DirNone
	case C.WM_LBUTTONDOWN, C.WM_MBUTTONDOWN, C.WM_RBUTTONDOWN:
		dir = mouse.DirPress
	case C.WM_LBUTTONUP, C.WM_MBUTTONUP, C.WM_RBUTTONUP:
		dir = mouse.DirRelease
	default:
		panic("sendMouseEvent() called on non-mouse message")
	}

	switch uMsg {
	case C.WM_MOUSEMOVE:
		button = mouse.ButtonNone
	case C.WM_LBUTTONDOWN, C.WM_LBUTTONUP:
		button = mouse.ButtonLeft
	case C.WM_MBUTTONDOWN, C.WM_MBUTTONUP:
		button = mouse.ButtonMiddle
	case C.WM_RBUTTONDOWN, C.WM_RBUTTONUP:
		button = mouse.ButtonRight
	}
	// TODO(andlabs): mouse wheel

	w.Send(mouse.Event{
		X:         float32(x),
		Y:         float32(y),
		Button:    button,
		Modifiers: keyModifiers(),
		Direction: dir,
	})
}

// Precondition: this is called in immediate response to the message that triggered the event (so not after w.Send).
func keyModifiers() (m key.Modifiers) {
	down := func(x C.int) bool {
		// GetKeyState gets the key state at the time of the message, so this is what we want.
		return C.GetKeyState(x)&0x80 != 0
	}

	if down(C.VK_CONTROL) {
		m |= key.ModControl
	}
	if down(C.VK_MENU) {
		m |= key.ModAlt
	}
	if down(C.VK_SHIFT) {
		m |= key.ModShift
	}
	if down(C.VK_LWIN) || down(C.VK_RWIN) {
		m |= key.ModMeta
	}
	return m
}
