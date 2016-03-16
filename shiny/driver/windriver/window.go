// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

// TODO: implement a back buffer.

import (
	"image"
	"image/color"
	"image/draw"
	"syscall"
	"unsafe"

	"golang.org/x/exp/shiny/driver/internal/drawer"
	"golang.org/x/exp/shiny/driver/internal/event"
	"golang.org/x/exp/shiny/driver/internal/win32"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

// Code in this package must follow general Windows rules about
// thread affinity of user interface objects:
//
// part 1: Window handles
// https://blogs.msdn.microsoft.com/oldnewthing/20051010-09/?p=33843
//
// part 2: Device contexts
// https://blogs.msdn.microsoft.com/oldnewthing/20051011-10/?p=33823
//
// part 3: Menus, icons, cursors, and accelerator tables
// https://blogs.msdn.microsoft.com/oldnewthing/20051012-00/?p=33803
//
// part 4: GDI objects and other notes on affinity
// https://blogs.msdn.microsoft.com/oldnewthing/20051013-11/?p=33783
//
// part 5: Object clean-up
// https://blogs.msdn.microsoft.com/oldnewthing/20051014-19/?p=33763

type windowImpl struct {
	hwnd syscall.Handle

	event.Queue

	sz             size.Event
	lifecycleStage lifecycle.Stage
}

func (w *windowImpl) Release() {
	win32.Release(w.hwnd)
}

var msgUpload = win32.AddWindowMsg(handleUpload)

func (w *windowImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle) {
	p := upload{
		dp:  dp,
		src: src.(*bufferImpl),
		sr:  sr,
	}
	win32.SendMessage(w.hwnd, msgUpload, 0, uintptr(unsafe.Pointer(&p)))
}

type upload struct {
	dp  image.Point
	src *bufferImpl
	sr  image.Rectangle
}

func handleUpload(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) {
	u := (*upload)(unsafe.Pointer(lParam))

	dc, err := win32.GetDC(hwnd)
	if err != nil {
		panic(err) // TODO handle errors
	}
	defer win32.ReleaseDC(hwnd, dc)

	// TODO(brainman): move preUpload / postUpload out of handleUpload,
	// because handleUpload can only be executed by one (message pump)
	// thread only
	u.src.preUpload()
	defer u.src.postUpload()

	// TODO: adjust if dp is outside dst bounds, or sr is outside src bounds.
	dr := u.sr.Add(u.dp.Sub(u.sr.Min))
	err = copyBitmapToDC(dc, dr, u.src.hbitmap, u.sr, draw.Src)
	if err != nil {
		panic(err) // TODO handle errors
	}
}

type handleWindowFillParams struct {
	dr    image.Rectangle
	color color.Color
	op    draw.Op
}

var msgWindowFill = win32.AddWindowMsg(handleWindowFill)

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	p := handleWindowFillParams{
		dr:    dr,
		color: src,
		op:    op,
	}
	win32.SendMessage(w.hwnd, msgWindowFill, 0, uintptr(unsafe.Pointer(&p)))
}

func handleWindowFill(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) {
	p := (*handleWindowFillParams)(unsafe.Pointer(lParam))

	dc, err := win32.GetDC(hwnd)
	if err != nil {
		panic(err) // TODO handle errors
	}
	defer win32.ReleaseDC(hwnd, dc)

	err = fill(dc, p.dr, p.color, p.op)
	if err != nil {
		panic(err) // TODO handle errors
	}
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	// TODO(brainman): use SetWorldTransform to implement generic Draw
}

type handleCopyParams struct {
	dp  image.Point
	src syscall.Handle
	sr  image.Rectangle
	op  draw.Op
}

var msgCopy = win32.AddWindowMsg(handleCopy)

func (w *windowImpl) Copy(dp image.Point, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	if op != draw.Src && op != draw.Over {
		drawer.Copy(w, dp, src, sr, op, opts)
		return
	}
	p := handleCopyParams{
		dp:  dp,
		src: src.(*textureImpl).bitmap,
		sr:  sr,
		op:  op,
	}
	win32.SendMessage(w.hwnd, msgCopy, 0, uintptr(unsafe.Pointer(&p)))
}

func handleCopy(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) {
	p := (*handleCopyParams)(unsafe.Pointer(lParam))

	dc, err := win32.GetDC(hwnd)
	if err != nil {
		panic(err) // TODO handle errors
	}
	defer win32.ReleaseDC(hwnd, dc)

	dr := p.sr.Add(p.dp.Sub(p.sr.Min))
	err = copyBitmapToDC(dc, dr, p.src, p.sr, p.op)
	if err != nil {
		panic(err) // TODO handle errors
	}
}

type handleScaleParams struct {
	dr  image.Rectangle
	src syscall.Handle
	sr  image.Rectangle
	op  draw.Op
}

var msgScale = win32.AddWindowMsg(handleScale)

func (w *windowImpl) Scale(dr image.Rectangle, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	if op != draw.Src && op != draw.Over {
		drawer.Scale(w, dr, src, sr, op, opts)
		return
	}
	p := handleScaleParams{
		dr:  dr,
		src: src.(*textureImpl).bitmap,
		sr:  sr,
		op:  op,
	}
	win32.SendMessage(w.hwnd, msgScale, 0, uintptr(unsafe.Pointer(&p)))
}

func handleScale(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) {
	p := (*handleScaleParams)(unsafe.Pointer(lParam))

	dc, err := win32.GetDC(hwnd)
	if err != nil {
		panic(err) // TODO handle errors
	}
	defer win32.ReleaseDC(hwnd, dc)

	err = copyBitmapToDC(dc, p.dr, p.src, p.sr, p.op)
	if err != nil {
		panic(err) // TODO handle errors
	}
}

func (w *windowImpl) Publish() screen.PublishResult {
	// TODO
	return screen.PublishResult{}
}

func init() {
	send := func(hwnd syscall.Handle, e interface{}) {
		theScreen.mu.Lock()
		w := theScreen.windows[hwnd]
		theScreen.mu.Unlock()

		w.Send(e)
	}
	win32.MouseEvent = func(hwnd syscall.Handle, e mouse.Event) { send(hwnd, e) }
	win32.PaintEvent = func(hwnd syscall.Handle, e paint.Event) { send(hwnd, e) }
	win32.KeyEvent = func(hwnd syscall.Handle, e key.Event) { send(hwnd, e) }
	win32.LifecycleEvent = lifecycleEvent
	win32.SizeEvent = sizeEvent
}

func lifecycleEvent(hwnd syscall.Handle, to lifecycle.Stage) {
	theScreen.mu.Lock()
	w := theScreen.windows[hwnd]
	theScreen.mu.Unlock()

	if w.lifecycleStage == to {
		return
	}
	w.Send(lifecycle.Event{
		From: w.lifecycleStage,
		To:   to,
	})
	w.lifecycleStage = to
}

func sizeEvent(hwnd syscall.Handle, e size.Event) {
	theScreen.mu.Lock()
	w := theScreen.windows[hwnd]
	theScreen.mu.Unlock()

	w.Send(e)

	if e != w.sz {
		w.sz = e
		w.Send(paint.Event{})
	}
}
