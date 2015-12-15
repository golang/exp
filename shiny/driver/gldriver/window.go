// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gldriver

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"golang.org/x/exp/shiny/driver/internal/pump"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/gl"
)

type windowImpl struct {
	s *screenImpl

	// id is a C data structure for the window.
	//	- For Cocoa, it's a ScreenGLView*.
	//	- For X11, it's a Window.
	id uintptr

	// ctx is a C data structure for the GL context.
	//	- For Cocoa, it's a NSOpenGLContext*.
	//	- For X11, it's an EGLSurface.
	ctx uintptr

	lifecycleStage lifecycle.Stage // current stage

	pump        pump.Pump
	publish     chan struct{}
	publishDone chan screen.PublishResult
	drawDone    chan struct{}

	// glctxMu is a mutex that enforces the atomicity of methods like
	// Texture.Upload or Window.Draw that are conceptually one operation
	// but are implemented by multiple OpenGL calls. OpenGL is a stateful
	// API, so interleaving OpenGL calls from separate higher-level
	// operations causes inconsistencies.
	glctxMu sync.Mutex
	glctx   gl.Context
	worker  gl.Worker

	szMu sync.Mutex
	sz   size.Event
}

func (w *windowImpl) Release() {
	// There are two ways a window can be closed. The first is the user
	// clicks the red button, in which case windowWillClose is called,
	// which calls Go's windowClosing, which does cleanup in
	// releaseCleanup below.
	//
	// The second way is Release is called programmatically. This calls
	// the NSWindow method performClose, which emulates the red button
	// being clicked.
	//
	// If these two approaches race, experiments suggest it is resolved
	// by performClose (which is called serially on the main thread).
	// If that stops being true, there is a check in windowWillClose
	// that avoids the Go cleanup code being invoked more than once.
	closeWindow(w.id)
}

func (w *windowImpl) releaseCleanup() {
	w.pump.Release()
}

func (w *windowImpl) Events() <-chan interface{} { return w.pump.Events() }
func (w *windowImpl) Send(event interface{})     { w.pump.Send(event) }

func (w *windowImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle) {
	// TODO: adjust if dp is outside dst bounds, or sr is outside src bounds.
	// TODO: keep a texture around for this purpose?
	t, err := w.s.NewTexture(sr.Size())
	if err != nil {
		panic(err)
	}
	t.Upload(dp, src, sr)
	w.Draw(f64.Aff3{
		1, 0, float64(dp.X),
		0, 1, float64(dp.Y),
	}, t, sr, draw.Src, nil)
	t.Release()
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	w.glctxMu.Lock()
	defer w.glctxMu.Unlock()

	if !w.glctx.IsProgram(w.s.fill.program) {
		p, err := compileProgram(w.glctx, fillVertexSrc, fillFragmentSrc)
		if err != nil {
			// TODO: initialize this somewhere else we can better handle the error.
			panic(err.Error())
		}
		w.s.fill.program = p
		w.s.fill.pos = w.glctx.GetAttribLocation(p, "pos")
		w.s.fill.mvp = w.glctx.GetUniformLocation(p, "mvp")
		w.s.fill.color = w.glctx.GetUniformLocation(p, "color")
		w.s.fill.quad = w.glctx.CreateBuffer()

		w.glctx.BindBuffer(gl.ARRAY_BUFFER, w.s.fill.quad)
		w.glctx.BufferData(gl.ARRAY_BUFFER, quadCoords, gl.STATIC_DRAW)
	}
	w.glctx.UseProgram(w.s.fill.program)

	dstL := float64(dr.Min.X)
	dstT := float64(dr.Min.Y)
	dstR := float64(dr.Max.X)
	dstB := float64(dr.Max.Y)
	writeAff3(w.glctx, w.s.fill.mvp, w.mvp(
		dstL, dstT,
		dstR, dstT,
		dstL, dstB,
	))

	r, g, b, a := src.RGBA()
	w.glctx.Uniform4f(
		w.s.fill.color,
		float32(r)/65535,
		float32(g)/65535,
		float32(b)/65535,
		float32(a)/65535,
	)

	w.glctx.BindBuffer(gl.ARRAY_BUFFER, w.s.fill.quad)
	w.glctx.EnableVertexAttribArray(w.s.fill.pos)
	w.glctx.VertexAttribPointer(w.s.fill.pos, 2, gl.FLOAT, false, 0, 0)

	w.glctx.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	w.glctx.DisableVertexAttribArray(w.s.fill.pos)
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	w.glctxMu.Lock()
	defer w.glctxMu.Unlock()

	w.glctx.UseProgram(w.s.texture.program)

	// Start with src-space left, top, right and bottom.
	srcL := float64(sr.Min.X)
	srcT := float64(sr.Min.Y)
	srcR := float64(sr.Max.X)
	srcB := float64(sr.Max.Y)
	// Transform to dst-space via the src2dst matrix, then to a MVP matrix.
	writeAff3(w.glctx, w.s.texture.mvp, w.mvp(
		src2dst[0]*srcL+src2dst[1]*srcT+src2dst[2],
		src2dst[3]*srcL+src2dst[4]*srcT+src2dst[5],
		src2dst[0]*srcR+src2dst[1]*srcT+src2dst[2],
		src2dst[3]*srcR+src2dst[4]*srcT+src2dst[5],
		src2dst[0]*srcL+src2dst[1]*srcB+src2dst[2],
		src2dst[3]*srcL+src2dst[4]*srcB+src2dst[5],
	))

	// OpenGL's fragment shaders' UV coordinates run from (0,0)-(1,1),
	// unlike vertex shaders' XY coordinates running from (-1,+1)-(+1,-1).
	//
	// We are drawing a rectangle PQRS, defined by two of its
	// corners, onto the entire texture. The two quads may actually
	// be equal, but in the general case, PQRS can be smaller.
	//
	//	(0,0) +---------------+ (1,0)
	//	      |  P +-----+ Q  |
	//	      |    |     |    |
	//	      |  S +-----+ R  |
	//	(0,1) +---------------+ (1,1)
	//
	// The PQRS quad is always axis-aligned. First of all, convert
	// from pixel space to texture space.
	t := src.(*textureImpl)
	tw := float64(t.size.X)
	th := float64(t.size.Y)
	px := float64(sr.Min.X-0) / tw
	py := float64(sr.Min.Y-0) / th
	qx := float64(sr.Max.X-0) / tw
	sy := float64(sr.Max.Y-0) / th
	// Due to axis alignment, qy = py and sx = px.
	//
	// The simultaneous equations are:
	//	  0 +   0 + a02 = px
	//	  0 +   0 + a12 = py
	//	a00 +   0 + a02 = qx
	//	a10 +   0 + a12 = qy = py
	//	  0 + a01 + a02 = sx = px
	//	  0 + a11 + a12 = sy
	writeAff3(w.glctx, w.s.texture.uvp, f64.Aff3{
		qx - px, 0, px,
		0, sy - py, py,
	})

	w.glctx.ActiveTexture(gl.TEXTURE0)
	w.glctx.BindTexture(gl.TEXTURE_2D, t.id)
	w.glctx.Uniform1i(w.s.texture.sample, 0)

	w.glctx.BindBuffer(gl.ARRAY_BUFFER, w.s.texture.quad)
	w.glctx.EnableVertexAttribArray(w.s.texture.pos)
	w.glctx.VertexAttribPointer(w.s.texture.pos, 2, gl.FLOAT, false, 0, 0)

	w.glctx.BindBuffer(gl.ARRAY_BUFFER, w.s.texture.quad)
	w.glctx.EnableVertexAttribArray(w.s.texture.inUV)
	w.glctx.VertexAttribPointer(w.s.texture.inUV, 2, gl.FLOAT, false, 0, 0)

	w.glctx.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	w.glctx.DisableVertexAttribArray(w.s.texture.pos)
	w.glctx.DisableVertexAttribArray(w.s.texture.inUV)
}

// mvp returns the Model View Projection matrix that maps the quadCoords unit
// square, (0, 0) to (1, 1), to a quad QV, such that QV in vertex shader space
// corresponds to the quad QP in pixel space, where QP is defined by three of
// its four corners - the arguments to this function. The three corners are
// nominally the top-left, top-right and bottom-left, but there is no
// constraint that e.g. tlx < trx.
//
// In pixel space, the window ranges from (0, 0) to (sz.WidthPx, sz.HeightPx).
// The Y-axis points downwards.
//
// In vertex shader space, the window ranges from (-1, +1) to (+1, -1), which
// is a 2-unit by 2-unit square. The Y-axis points upwards.
func (w *windowImpl) mvp(tlx, tly, trx, try, blx, bly float64) f64.Aff3 {
	w.szMu.Lock()
	sz := w.sz
	w.szMu.Unlock()

	// Convert from pixel coords to vertex shader coords.
	invHalfWidth := +2 / float64(sz.WidthPx)
	invHalfHeight := -2 / float64(sz.HeightPx)
	tlx = tlx*invHalfWidth - 1
	tly = tly*invHalfHeight + 1
	trx = trx*invHalfWidth - 1
	try = try*invHalfHeight + 1
	blx = blx*invHalfWidth - 1
	bly = bly*invHalfHeight + 1

	// The resultant affine matrix:
	//	- maps (0, 0) to (tlx, tly).
	//	- maps (1, 0) to (trx, try).
	//	- maps (0, 1) to (blx, bly).
	return f64.Aff3{
		trx - tlx, blx - tlx, tlx,
		try - tly, bly - tly, tly,
	}
}

func (w *windowImpl) Publish() screen.PublishResult {
	// gl.Flush is a lightweight (on modern GL drivers) blocking call
	// that ensures all GL functions pending in the gl package have
	// been passed onto the GL driver before the app package attempts
	// to swap the screen buffer.
	//
	// This enforces that the final receive (for this paint cycle) on
	// gl.WorkAvailable happens before the send on publish.
	w.glctxMu.Lock()
	w.glctx.Flush()
	w.glctxMu.Unlock()

	w.publish <- struct{}{}
	res := <-w.publishDone

	select {
	case w.drawDone <- struct{}{}:
	default:
	}

	return res
}
