// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"image"
	"image/draw"
	"sync"

	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
)

const textureDepth = 32

type textureImpl struct {
	s *screenImpl

	size image.Point
	xm   xproto.Pixmap
	xp   render.Picture

	mu       sync.Mutex
	released bool
}

func (t *textureImpl) Size() image.Point { return t.size }

func (t *textureImpl) Release() {
	t.mu.Lock()
	released := t.released
	t.released = true
	t.mu.Unlock()

	if released {
		return
	}
	render.FreePicture(t.s.xc, t.xp)
	xproto.FreePixmap(t.s.xc, t.xm)
}

func (t *textureImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle, sender screen.Sender) {
	src.(*bufferImpl).upload(t, xproto.Drawable(t.xm), t.s.gcontext32, textureDepth, dp, sr, sender)
}

func (t *textureImpl) draw(xp render.Picture, src2dst *f64.Aff3, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	renderOp := uint8(render.PictOpOver)
	if op == draw.Src {
		renderOp = render.PictOpSrc
	}

	// TODO: honor all of src2dst, not just the translation.
	dstX := int(src2dst[2]) - sr.Min.X
	dstY := int(src2dst[5]) - sr.Min.Y

	render.Composite(t.s.xc, renderOp, t.xp, 0, xp,
		int16(sr.Min.X), int16(sr.Min.Y), // SrcX, SrcY,
		0, 0, // MaskX, MaskY,
		int16(dstX), int16(dstY), // DstX, DstY,
		uint16(sr.Dx()), uint16(sr.Dy()), // Width, Height,
	)
}
