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
	"sync"
	"unsafe"

	"golang.org/x/exp/shiny/driver/internal/pump"
	"golang.org/x/exp/shiny/driver/internal/win32"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

var (
	uploadsMu sync.Mutex
	uploads   = map[uintptr]upload{}
	uploadID  uintptr
)

type windowImpl struct {
	hwnd win32.HWND
	pump pump.Pump

	sz             size.Event
	lifecycleStage lifecycle.Stage
}

func (w *windowImpl) Release() {
	win32.Release(w.hwnd)
	w.pump.Release()
}

func (w *windowImpl) Events() <-chan interface{} { return w.pump.Events() }
func (w *windowImpl) Send(event interface{})     { w.pump.Send(event) }

func (w *windowImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle) {
	completion := make(chan struct{})

	// Protect struct contents from being GCed
	uploadsMu.Lock()
	uploadID++
	id := uploadID
	uploads[id] = upload{
		dp:         dp,
		src:        src.(*bufferImpl),
		sr:         sr,
		completion: completion,
	}
	uploadsMu.Unlock()

	win32.SendMessage(w.hwnd, msgUpload, id, 0)

	<-completion
}

type upload struct {
	dp         image.Point
	src        *bufferImpl
	sr         image.Rectangle
	completion chan struct{}
}

func handleUpload(hwnd win32.HWND, uMsg uint32, wParam, lParam uintptr) {
	id := wParam
	uploadsMu.Lock()
	u := uploads[id]
	delete(uploads, id)
	uploadsMu.Unlock()

	dc, err := win32.GetDC(hwnd)
	if err != nil {
		panic(err) // TODO handle errors
	}
	defer win32.ReleaseDC(hwnd, dc)

	u.src.preUpload(true)

	// TODO: adjust if dp is outside dst bounds, or sr is outside src bounds.
	err = blit(dc, _POINT{int32(u.dp.X), int32(u.dp.Y)}, u.src.hbitmap, &_RECT{
		Left:   int32(u.sr.Min.X),
		Top:    int32(u.sr.Min.Y),
		Right:  int32(u.sr.Max.X),
		Bottom: int32(u.sr.Max.Y),
	})
	go func() {
		u.src.postUpload()
		close(u.completion)
	}()
	if err != nil {
		panic(err) // TODO handle errors
	}
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
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
	win32.SendMessage(w.hwnd, msg, uintptr(color), uintptr(unsafe.Pointer(&rect)))
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	// TODO
}

func (w *windowImpl) Publish() screen.PublishResult {
	// TODO
	return screen.PublishResult{}
}

func init() {
	send := func(hwnd win32.HWND, e interface{}) {
		theScreen.mu.Lock()
		w := theScreen.windows[hwnd]
		theScreen.mu.Unlock()

		w.Send(e)
	}
	win32.MouseEvent = func(hwnd win32.HWND, e mouse.Event) { send(hwnd, e) }
	win32.PaintEvent = func(hwnd win32.HWND, e paint.Event) { send(hwnd, e) }
	win32.KeyEvent = func(hwnd win32.HWND, e key.Event) { send(hwnd, e) }
	win32.LifecycleEvent = lifecycleEvent
	win32.SizeEvent = sizeEvent
}

func lifecycleEvent(hwnd win32.HWND, to lifecycle.Stage) {
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

func sizeEvent(hwnd win32.HWND, e size.Event) {
	theScreen.mu.Lock()
	w := theScreen.windows[hwnd]
	theScreen.mu.Unlock()

	w.Send(e)

	if e != w.sz {
		w.sz = e
		w.Send(paint.Event{})
	}
}
