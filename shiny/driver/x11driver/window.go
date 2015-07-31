// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"image"
	"image/draw"
	"log"
	"sync"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/exp/shiny/driver/internal/pump"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
)

type windowImpl struct {
	s *screenImpl

	xw xproto.Window
	xg xproto.Gcontext
	xp render.Picture

	pump    pump.Pump
	xevents chan xgb.Event

	mu       sync.Mutex
	released bool
}

func (w *windowImpl) run() {
	for {
		select {
		// TODO: things other than X11 events.

		case ev, ok := <-w.xevents:
			if !ok {
				return
			}
			switch ev := ev.(type) {
			default:
				// TODO: implement.
				log.Println(ev)
			}
		}
	}
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
	// TODO: call render.FreePicture.
	xproto.FreeGC(w.s.xc, w.xg)
	xproto.DestroyWindow(w.s.xc, w.xw)
	w.pump.Release()
}

func (w *windowImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle, sender screen.Sender) {
	b := src.(*bufferImpl)
	b.preUpload()

	// TODO: adjust if dp is outside dst bounds, or sr is outside src bounds.
	dr := sr.Sub(sr.Min).Add(dp)

	cookie := shm.PutImageChecked(
		w.s.xc, xproto.Drawable(w.xw), w.xg,
		uint16(b.size.X), uint16(b.size.Y), // TotalWidth, TotalHeight,
		uint16(sr.Min.X), uint16(sr.Min.Y), // SrcX, SrcY,
		uint16(dr.Dx()), uint16(dr.Dy()), // SrcWidth, SrcHeight,
		int16(dr.Min.X), int16(dr.Min.Y), // DstX, DstY,
		w.s.xsi.RootDepth, xproto.ImageFormatZPixmap,
		1, b.xs, 0, // 1 means send a completion event, 0 means a zero offset.
	)

	w.s.mu.Lock()
	w.s.uploads[cookie.Sequence] = completion{
		sender: sender,
		event: screen.UploadedEvent{
			Buffer:   src,
			Uploader: w,
		},
	}
	w.s.mu.Unlock()
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	// TODO.
}

func (w *windowImpl) EndPaint() {
	// TODO.
}
