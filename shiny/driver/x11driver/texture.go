// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"fmt"
	"image"
	"image/color"
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

func (t *textureImpl) Size() image.Point       { return t.size }
func (t *textureImpl) Bounds() image.Rectangle { return image.Rectangle{Max: t.size} }

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

func (t *textureImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	fill(t.s.xc, t.xp, dr, src, op)
}

func f64ToFixed(x float64) render.Fixed {
	return render.Fixed(x * 65536)
}

func inv(x *f64.Aff3) *f64.Aff3 {
	return &f64.Aff3{
		x[4] / (x[0]*x[4] - x[1]*x[3]),
		x[1] / (x[1]*x[3] - x[0]*x[4]),
		(x[2]*x[4] - x[1]*x[5]) / (x[1]*x[3] - x[0]*x[4]),
		x[3] / (x[1]*x[3] - x[0]*x[4]),
		x[0] / (x[0]*x[4] - x[1]*x[3]),
		(x[2]*x[3] - x[0]*x[5]) / (x[0]*x[4] - x[1]*x[3]),
	}
}

func (t *textureImpl) draw(xp render.Picture, src2dst *f64.Aff3, sr image.Rectangle, op draw.Op, w, h int, opts *screen.DrawOptions) {
	// TODO: honor sr.Max

	// The XTransform matrix maps from destination pixels to source
	// pixels, so we invert src2dst.
	dst2src := inv(src2dst)
	err := render.SetPictureTransformChecked(t.s.xc, t.xp, render.Transform{
		f64ToFixed(dst2src[0]), f64ToFixed(dst2src[1]), f64ToFixed(dst2src[2]),
		f64ToFixed(dst2src[3]), f64ToFixed(dst2src[4]), f64ToFixed(dst2src[5]),
		f64ToFixed(0), f64ToFixed(0), f64ToFixed(1),
	}).Check()

	if err != nil {
		panic(fmt.Errorf("x11driver: cannot transform picture: %v", err))
	}
	err = render.SetPictureFilterChecked(t.s.xc, t.xp, uint16(len("bilinear")), "bilinear", nil).Check()
	if err != nil {
		panic(fmt.Errorf("x11driver: cannot filter picture: %v", err))
	}

	render.Composite(t.s.xc, renderOp(op), t.xp, 0, xp,
		int16(sr.Min.X), int16(sr.Min.Y), // SrcX, SrcY,
		0, 0, // MaskX, MaskY,
		0, 0, // DstX, DstY,
		uint16(w), uint16(h), // Width, Height,
	)
}

func renderOp(op draw.Op) byte {
	if op == draw.Src {
		return render.PictOpSrc
	}
	return render.PictOpOver
}
