// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"image"
	"image/color"
	"image/draw"
	"math"
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

	// renderMu is a mutex that enforces the atomicity of methods like
	// Window.Draw that are conceptually one operation but are implemented by
	// multiple X11/Render calls. X11/Render is a stateful API, so interleaving
	// X11/Render calls from separate higher-level operations causes
	// inconsistencies.
	renderMu sync.Mutex

	releasedMu sync.Mutex
	released   bool
}

func (t *textureImpl) Size() image.Point       { return t.size }
func (t *textureImpl) Bounds() image.Rectangle { return image.Rectangle{Max: t.size} }

func (t *textureImpl) Release() {
	t.releasedMu.Lock()
	released := t.released
	t.released = true
	t.releasedMu.Unlock()

	if released {
		return
	}
	render.FreePicture(t.s.xc, t.xp)
	xproto.FreePixmap(t.s.xc, t.xm)
}

func (t *textureImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle) {
	src.(*bufferImpl).upload(t, xproto.Drawable(t.xm), t.s.gcontext32, textureDepth, dp, sr)
}

func (t *textureImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	fill(t.s.xc, t.xp, dr, src, op)
}

// f64ToFixed converts from float64 to X11/Render's 16.16 fixed point.
func f64ToFixed(x float64) render.Fixed {
	return render.Fixed(x * 65536)
}

func inv(x *f64.Aff3) f64.Aff3 {
	invDet := 1 / (x[0]*x[4] - x[1]*x[3])
	return f64.Aff3{
		+x[4] * invDet,
		-x[1] * invDet,
		(x[1]*x[5] - x[2]*x[4]) * invDet,
		-x[3] * invDet,
		+x[0] * invDet,
		(x[2]*x[3] - x[0]*x[5]) * invDet,
	}
}

func (t *textureImpl) draw(xp render.Picture, src2dst *f64.Aff3, sr image.Rectangle, op draw.Op, w, h int, opts *screen.DrawOptions) {
	// TODO: honor sr.Max

	t.renderMu.Lock()
	defer t.renderMu.Unlock()

	// For simple copies and scales, the inverse matrix is trivial to compute,
	// and we do not need the "Src becomes OutReverse plus Over" dance (see
	// below). Thus, draw can be one render.SetPictureTransform call and then
	// one render.Composite call, regardless of whether or not op is Src.
	if src2dst[1] == 0 && src2dst[3] == 0 {
		dstXMin := float64(sr.Min.X)*src2dst[0] + src2dst[2]
		dstXMax := float64(sr.Max.X)*src2dst[0] + src2dst[2]
		if dstXMin > dstXMax {
			// TODO: check if this (and below) works when src2dst[0] < 0.
			dstXMin, dstXMax = dstXMax, dstXMin
		}
		dXMin := int(math.Floor(dstXMin))
		dXMax := int(math.Ceil(dstXMax))

		dstYMin := float64(sr.Min.Y)*src2dst[4] + src2dst[5]
		dstYMax := float64(sr.Max.Y)*src2dst[4] + src2dst[5]
		if dstYMin > dstYMax {
			// TODO: check if this (and below) works when src2dst[4] < 0.
			dstYMin, dstYMax = dstYMax, dstYMin
		}
		dYMin := int(math.Floor(dstYMin))
		dYMax := int(math.Ceil(dstYMax))

		render.SetPictureTransform(t.s.xc, t.xp, render.Transform{
			f64ToFixed(1 / src2dst[0]), 0, 0,
			0, f64ToFixed(1 / src2dst[4]), 0,
			0, 0, 1 << 16,
		})
		render.Composite(t.s.xc, renderOp(op), t.xp, 0, xp,
			int16(sr.Min.X), int16(sr.Min.Y), // SrcX, SrcY,
			0, 0, // MaskX, MaskY,
			int16(dXMin), int16(dYMin), // DstX, DstY,
			uint16(dXMax-dXMin), uint16(dYMax-dYMin), // Width, Height,
		)
		return
	}

	// The X11/Render transform matrix maps from destination pixels to source
	// pixels, so we invert src2dst.
	dst2src := inv(src2dst)
	render.SetPictureTransform(t.s.xc, t.xp, render.Transform{
		f64ToFixed(dst2src[0]), f64ToFixed(dst2src[1]), f64ToFixed(dst2src[2]),
		f64ToFixed(dst2src[3]), f64ToFixed(dst2src[4]), f64ToFixed(dst2src[5]),
		0, 0, 1 << 16,
	})

	if op == draw.Src {
		// render.Composite visits every dst-space pixel in the rectangle
		// defined by its args DstX, DstY, Width, Height. That axis-aligned
		// bounding box (AABB) must contain the transformation of the sr
		// rectangle in src-space to a quad in dst-space, but it need not be
		// the smallest possible AABB.
		//
		// In any case, for arbitrary src2dst affine transformations, which
		// include rotations, this means that a naive render.Composite call
		// will affect those pixels inside the AABB but outside the quad. For
		// the draw.Src operator, this means that pixels in that AABB can be
		// incorrectly set to zero.
		//
		// Instead, we implement the draw.Src operator as two render.Composite
		// calls. The first one (using the PictOpOutReverse operator) clears
		// the dst-space quad but leaves pixels outside that quad (but inside
		// the AABB) untouched. The second one (using the PictOpOver operator)
		// fills in the quad and again does not touch the pixels outside.
		//
		// What X11/Render calls PictOpOutReverse is also known as dst-out. See
		// http://www.w3.org/TR/SVGCompositing/examples/compop-porterduff-examples.png
		// for a visualization.
		//
		// The arguments to this render.Composite call are identical to the
		// second one call below, other than the compositing operator.
		//
		// TODO: the source picture for this call needs to be fully opaque even
		// if t.xp isn't.
		render.Composite(t.s.xc, render.PictOpOutReverse, t.xp, 0, xp,
			int16(sr.Min.X), int16(sr.Min.Y), 0, 0, 0, 0, uint16(w), uint16(h),
		)
	}

	// TODO: tighten the (0, 0)-(w, h) dst rectangle. As it is, we're
	// compositing an unnecessarily large number of pixels.

	render.Composite(t.s.xc, render.PictOpOver, t.xp, 0, xp,
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
