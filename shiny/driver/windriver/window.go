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
	"math"
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

type handleDrawParams struct {
	src2dst f64.Aff3
	src     syscall.Handle
	sr      image.Rectangle
	op      draw.Op
}

var msgDraw = win32.AddWindowMsg(handleDraw)

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	if op != draw.Src && op != draw.Over {
		// TODO:
		return
	}
	p := handleDrawParams{
		src2dst: src2dst,
		src:     src.(*textureImpl).bitmap,
		sr:      sr,
		op:      op,
	}
	win32.SendMessage(w.hwnd, msgDraw, 0, uintptr(unsafe.Pointer(&p)))
}

func handleDraw(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) {
	p := (*handleDrawParams)(unsafe.Pointer(lParam))
	err := drawWindow(hwnd, p.src2dst, p.src, p.sr, p.op)
	if err != nil {
		panic(err) // TODO handle errors
	}
}

func drawWindow(hwnd syscall.Handle, src2dst f64.Aff3, src syscall.Handle, sr image.Rectangle, op draw.Op) (retErr error) {
	dc, err := win32.GetDC(hwnd)
	if err != nil {
		return err
	}
	defer win32.ReleaseDC(hwnd, dc)

	var dr image.Rectangle
	if src2dst[1] != 0 || src2dst[3] != 0 {
		// general drawing
		dr = sr.Sub(sr.Min)

		prevmode, err := _SetGraphicsMode(dc, _GM_ADVANCED)
		if err != nil {
			return err
		}
		defer func() {
			_, err := _SetGraphicsMode(dc, prevmode)
			if retErr == nil {
				retErr = err
			}
		}()

		x := _XFORM{
			eM11: +float32(src2dst[0]),
			eM12: -float32(src2dst[1]),
			eM21: -float32(src2dst[3]),
			eM22: +float32(src2dst[4]),
			eDx:  +float32(src2dst[2]),
			eDy:  +float32(src2dst[5]),
		}
		err = _SetWorldTransform(dc, &x)
		if err != nil {
			return err
		}
		defer func() {
			err := _ModifyWorldTransform(dc, nil, _MWT_IDENTITY)
			if retErr == nil {
				retErr = err
			}
		}()
	} else if src2dst[0] == 1 && src2dst[4] == 1 {
		// copy bitmap
		dp := image.Point{int(src2dst[2]), int(src2dst[5])}
		dr = sr.Add(dp.Sub(sr.Min))
	} else {
		// scale bitmap
		dstXMin := float64(sr.Min.X)*src2dst[0] + src2dst[2]
		dstXMax := float64(sr.Max.X)*src2dst[0] + src2dst[2]
		if dstXMin > dstXMax {
			// TODO: check if this (and below) works when src2dst[0] < 0.
			dstXMin, dstXMax = dstXMax, dstXMin
		}
		dstYMin := float64(sr.Min.Y)*src2dst[4] + src2dst[5]
		dstYMax := float64(sr.Max.Y)*src2dst[4] + src2dst[5]
		if dstYMin > dstYMax {
			// TODO: check if this (and below) works when src2dst[4] < 0.
			dstYMin, dstYMax = dstYMax, dstYMin
		}
		dr = image.Rectangle{
			image.Point{int(math.Floor(dstXMin)), int(math.Floor(dstYMin))},
			image.Point{int(math.Ceil(dstXMax)), int(math.Ceil(dstYMax))},
		}
	}
	return copyBitmapToDC(dc, dr, src, sr, op)
}

func (w *windowImpl) Copy(dp image.Point, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	drawer.Copy(w, dp, src, sr, op, opts)
}

func (w *windowImpl) Scale(dr image.Rectangle, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	drawer.Scale(w, dr, src, sr, op, opts)
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
