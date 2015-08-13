// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package windriver

// #include "windriver.h"
import "C"

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/exp/shiny/driver/internal/pump"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/paint"
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
	return &window{
		hwnd: hwnd,
		pump: pump.Make(),
	}, nil
}

func (w *window) Release() {
	if w.hwnd == nil { // already released?
		return
	}
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
