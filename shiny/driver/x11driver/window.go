// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"image"
	"image/draw"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
)

type windowImpl struct {
	s *screenImpl

	xw xproto.Window
	xg xproto.Gcontext
	xp render.Picture

	xevents chan xgb.Event
}

func (w *windowImpl) run() {
	for {
		select {
		// TODO: things other than X11 events.

		case ev := <-w.xevents:
			// TODO: implement.
			log.Println(ev)
		}
	}
}

func (w *windowImpl) Release() {
	// TODO.
}

func (w *windowImpl) Events() <-chan interface{} {
	// TODO.
	return nil
}

func (w *windowImpl) Send(event interface{}) {
	// TODO.
}

func (w *windowImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle) {
	// TODO.
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	// TODO.
}

func (w *windowImpl) EndPaint() {
	// TODO.
}
