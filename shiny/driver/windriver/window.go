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
	"syscall"
	"unsafe"

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
	windows     = map[syscall.Handle]*window{}
	windowsLock sync.Mutex
)

type window struct {
	hwnd syscall.Handle
	pump pump.Pump
}

func newWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	hwnd, err := createWindow()
	if err != nil {
		return nil, err
	}

	w := &window{
		hwnd: hwnd,
		pump: pump.Make(),
	}

	windowsLock.Lock()
	windows[hwnd] = w
	windowsLock.Unlock()

	// Send a fake size event.
	// Windows won't generate the WM_WINDOWPOSCHANGED
	// we trigger a resize on for the initial size, so we have to do
	// it ourselves. The example/basic program assumes it will
	// receive a size.Event for the initial window size that isn't 0x0.
	var r _RECT
	// TODO(andlabs) error check
	_GetClientRect(w.hwnd, &r)
	sendSizeEvent(w.hwnd, &r)

	return w, nil
}

func (w *window) Release() {
	if w.hwnd == 0 { // already released?
		return
	}

	windowsLock.Lock()
	delete(windows, w.hwnd)
	windowsLock.Unlock()

	// TODO(andlabs): check for errors from this?
	// TODO(andlabs): remove unsafe
	_DestroyWindow(syscall.Handle(uintptr(unsafe.Pointer(w.hwnd))))
	w.hwnd = 0
	w.pump.Release()

	// TODO(andlabs): what happens if we're still painting?
}

func (w *window) Events() <-chan interface{} { return w.pump.Events() }
func (w *window) Send(event interface{})     { w.pump.Send(event) }

func (w *window) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle, sender screen.Sender) {
	// TODO
}

func (w *window) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	rect := _RECT{
		Left:   int32(dr.Min.X),
		Top:    int32(dr.Min.Y),
		Right:  int32(dr.Max.X),
		Bottom: int32(dr.Max.Y),
	}
	r, g, b, a := src.RGBA()
	r >>= 8
	g >>= 8
	b >>= 8
	a >>= 8
	color := (a << 24) | (r << 16) | (g << 8) | b
	msg := uint32(msgFillOver)
	if op == draw.Src {
		msg = msgFillSrc
	}
	// Note: this SendMessage won't return until after the fill
	// completes, so using &rect is safe.
	_SendMessage(w.hwnd, msg, uintptr(color), uintptr(unsafe.Pointer(&rect)))
}

func (w *window) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	// TODO
}

func (w *window) EndPaint(p paint.Event) {
	// TODO
}

func handlePaint(hwnd syscall.Handle) {
	windowsLock.Lock()
	w := windows[hwnd]
	windowsLock.Unlock()

	// TODO(andlabs) - this won't be necessary after the Go rewrite
	// Windows sends spurious WM_PAINT messages at window
	// creation.
	if w == nil {
		return
	}

	w.Send(paint.Event{}) // TODO(andlabs): fill struct field
}

func sendSizeEvent(hwnd syscall.Handle, r *_RECT) {
	windowsLock.Lock()
	w := windows[hwnd]
	windowsLock.Unlock()

	width := int(r.Right - r.Left)
	height := int(r.Bottom - r.Top)
	// TODO(andlabs): don't assume that PixelsPerPt == 1
	w.Send(size.Event{
		WidthPx:     width,
		HeightPx:    height,
		WidthPt:     geom.Pt(width),
		HeightPt:    geom.Pt(height),
		PixelsPerPt: 1,
	})
}

func sendMouseEvent(hwnd syscall.Handle, uMsg uint32, x int32, y int32) {
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

func windowWndProc(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	switch uMsg {
	case _WM_PAINT:
		handlePaint(hwnd)
		// defer to DefWindowProc; it will handle validation for us
		return _DefWindowProc(hwnd, uMsg, wParam, lParam)
	case _WM_WINDOWPOSCHANGED:
		wp := (*_WINDOWPOS)(unsafe.Pointer(lParam))
		if wp.Flags&_SWP_NOSIZE != 0 {
			break
		}
		var r _RECT
		if _GetClientRect(hwnd, &r) != nil {
			// TODO(andlabs)
		}
		sendSizeEvent(hwnd, &r)
		return 0
	case _WM_MOUSEMOVE, _WM_LBUTTONDOWN:
		// TODO(andlabs): call SetFocus()?
	case _WM_LBUTTONUP, _WM_MBUTTONDOWN, _WM_MBUTTONUP, _WM_RBUTTONDOWN, _WM_RBUTTONUP:
		sendMouseEvent(hwnd, uMsg, _GET_X_LPARAM(lParam), _GET_Y_LPARAM(lParam))
		return 0
	case _WM_KEYDOWN, _WM_KEYUP, _WM_SYSKEYDOWN, _WM_SYSKEYUP:
		// TODO
	case msgFillSrc:
		// TODO error checks
		dc, err := _GetDC(hwnd)
		if err != nil {
			// TODO handle errors
			break
		}
		r := (*_RECT)(unsafe.Pointer(lParam))
		// TODO handle errors
		fillSrc(dc, r, _COLORREF(wParam))
		_ReleaseDC(hwnd, dc)
	case msgFillOver:
		// TODO error checks
		dc, err := _GetDC(hwnd)
		if err != nil {
			// TODO handle errors
			break
		}
		r := (*_RECT)(unsafe.Pointer(lParam))
		// TODO handle errors
		fillOver(dc, r, _COLORREF(wParam))
		_ReleaseDC(hwnd, dc)
	}
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}

const windowClass = "shiny_Window"

func initWindowClass() (err error) {
	wcname, err := syscall.UTF16PtrFromString(windowClass)
	if err != nil {
		return err
	}
	_, err = _RegisterClass(&_WNDCLASS{
		LpszClassName: wcname,
		LpfnWndProc:   syscall.NewCallback(windowWndProc),
		HIcon:         hDefaultIcon,
		HCursor:       hDefaultCursor,
		HInstance:     hThisInstance,
		// TODO(andlabs): change this to something else? NULL? the hollow brush?
		HbrBackground: syscall.Handle(_COLOR_BTNFACE + 1),
	})
	return err
}

func createWindow() (syscall.Handle, error) {
	// TODO(brainman): convert windowClass to *uint16 once (in initWindowClass)
	wcname, err := syscall.UTF16PtrFromString(windowClass)
	if err != nil {
		return 0, err
	}
	title, err := syscall.UTF16PtrFromString("Shiny Window")
	if err != nil {
		return 0, err
	}
	h, err := _CreateWindowEx(0,
		wcname, title,
		_WS_OVERLAPPEDWINDOW,
		_CW_USEDEFAULT, _CW_USEDEFAULT,
		_CW_USEDEFAULT, _CW_USEDEFAULT,
		0, 0, hThisInstance, 0)
	if err != nil {
		return 0, err
	}
	// TODO(andlabs): use proper nCmdShow
	_ShowWindow(h, _SW_SHOWDEFAULT)
	// TODO(andlabs): call UpdateWindow()
	return h, nil
}
