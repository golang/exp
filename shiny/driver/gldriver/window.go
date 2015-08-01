// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build 386 amd64

package gldriver

import (
	"image"
	"image/draw"
	"sync"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/config"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/gl"
)

type windowImpl struct {
	s  *screenImpl
	id uintptr // *C.ScreenGLView

	eventsIn  chan interface{}
	eventsOut chan interface{}
	endPaint  chan paint.Event

	draw     chan struct{}
	drawDone chan struct{}

	mu  sync.Mutex
	cfg config.Event
}

type stopPumping struct{}

// pump forwards events from eventsIn to eventsOut.
//
// All events will eventually send, in order, but eventsIn will always
// be ready to send/receive soon, even if eventsOut currently isn't.
// It is effectively an infinitely buffered channel.
//
// In particular, goroutine A sending on eventsIn will not deadlock
// even if goroutine B that's responsible for receiving on eventsOut
// is currently blocked trying to send to A on a separate channel.
//
// Send a stopPumping on the eventsIn channel to close the eventsOut
// channel after all queued events are sent on eventsOut. After that,
// other goroutines can still send to eventsIn, so that such sends
// won't block forever, but such events will be ignored.
func (w *windowImpl) pump() {
	// initialSize is the initial size of the circular buffer. It must be a
	// power of 2.
	const initialSize = 16
	i, j, buf, mask := 0, 0, make([]interface{}, initialSize), initialSize-1

	maybeSrc := w.eventsIn
	for {
		maybeDst := w.eventsOut
		if i == j {
			maybeDst = nil
		}
		if maybeDst == nil && maybeSrc == nil {
			break
		}

		select {
		case maybeDst <- buf[i&mask]:
			buf[i&mask] = nil
			i++

		case e := <-maybeSrc:
			if _, ok := e.(stopPumping); ok {
				maybeSrc = nil
				continue
			}

			// Allocate a bigger buffer if necessary.
			if i+len(buf) == j {
				b := make([]interface{}, 2*len(buf))
				n := copy(b, buf[j&mask:])
				copy(b[n:], buf[:j&mask])
				i, j = 0, len(buf)
				buf, mask = b, len(b)-1
			}

			buf[j&mask] = e
			j++
		}
	}

	close(w.eventsOut)
	// Block forever.
	for range w.eventsIn {
	}
}

func (w *windowImpl) Release() {
	// TODO.
}

func (w *windowImpl) Events() <-chan interface{} {
	return w.eventsOut
}

func (w *windowImpl) Send(event interface{}) {
	w.eventsIn <- event
}

func (w *windowImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle, sender screen.Sender) {
	// TODO: adjust if dp is outside dst bounds, or sr is outside src bounds.
	// TODO: keep a texture around for this purpose?
	t, err := w.s.NewTexture(sr.Size())
	if err != nil {
		panic(err)
	}
	t.Upload(dp, src, sr, sender)
	w.Draw(f64.Aff3{1, 0, 0, 0, 1, 0}, t, sr, draw.Src, nil)
	t.Release()
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	t := src.(*textureImpl)

	w.mu.Lock()
	cfg := w.cfg
	w.mu.Unlock()

	size := sr.Size()
	tx, ty := float64(size.X), float64(size.Y)
	wx, wy := float64(cfg.WidthPx), float64(cfg.HeightPx)
	rx, ry := tx/wx, ty/wy

	// We are drawing the texture src onto the window's framebuffer.
	// The texture is (0,0)-(tx,ty). The window is (0,0)-(wx,wy), which
	// in vertex shader space is
	//
	//	(-1, +1) (+1, +1)
	//	(-1, -1) (+1, -1)
	//
	// A src2dst unit affine transform
	//
	// 	1 0 0
	// 	0 1 0
	// 	0 0 1
	//
	// should result in a (tx,ty) texture appearing in the upper-left
	// (tx, ty) pixels of the window.
	//
	// Setting w.s.texture.mvp to a unit affine transform results in
	// mapping the 2-unit square (-1,+1)-(+1,-1) given by quadXYCoords
	// in texture.go to the same coordinates in vertex shader space.
	// Thus, it results in the whole texture ((tx, ty) in texture
	// space) occupying the whole window ((wx, wy) in window space).
	//
	// A scaling affine transform
	//
	//	rx  0  0
	//	 0 ry  0
	//	 0  0  1
	//
	// results in a (tx, ty) texture occupying (tx, ty) pixels in the
	// center of the window.
	//
	// For upper-left alignment, we want to translate by
	// (-(1-rx), 1-ry), which is the affine transform
	//
	//	1    0   -1+rx
	//	0    1   +1-ry
	//	0    0       1
	//
	// These multiply to give:
	a := f64.Aff3{
		rx, 0, -1 + rx,
		0, ry, +1 - ry,
	}

	gl.UseProgram(w.s.texture.program)
	writeAff3(w.s.texture.mvp, mul(a, src2dst))

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
	writeAff3(w.s.texture.uvp, f64.Aff3{
		qx - px, 0, px,
		0, sy - py, py,
	})

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.Uniform1i(w.s.texture.sample, 0)

	gl.BindBuffer(gl.ARRAY_BUFFER, w.s.texture.quadXY)
	gl.EnableVertexAttribArray(w.s.texture.pos)
	gl.VertexAttribPointer(w.s.texture.pos, 2, gl.FLOAT, false, 0, 0)

	gl.BindBuffer(gl.ARRAY_BUFFER, w.s.texture.quadUV)
	gl.EnableVertexAttribArray(w.s.texture.inUV)
	gl.VertexAttribPointer(w.s.texture.inUV, 2, gl.FLOAT, false, 0, 0)

	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.DisableVertexAttribArray(w.s.texture.pos)
	gl.DisableVertexAttribArray(w.s.texture.inUV)
}

func (w *windowImpl) EndPaint(e paint.Event) {
	// gl.Flush is a lightweight (on modern GL drivers) blocking call
	// that ensures all GL functions pending in the gl package have
	// been passed onto the GL driver before the app package attempts
	// to swap the screen buffer.
	//
	// This enforces that the final receive (for this paint cycle) on
	// gl.WorkAvailable happens before the send on endPaint.
	gl.Flush()
	w.endPaint <- e
}
