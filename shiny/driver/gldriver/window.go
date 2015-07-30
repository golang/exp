// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build 386 amd64

package gldriver

import (
	"image"
	"image/draw"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
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
	// TODO.
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src screen.Texture, sr image.Rectangle, op draw.Op, opts *screen.DrawOptions) {
	// TODO.
}

func (w *windowImpl) EndPaint() {
	// gl.Flush is a lightweight (on modern GL drivers) blocking call
	// that ensures all GL functions pending in the gl package have
	// been passed onto the GL driver before the app package attempts
	// to swap the screen buffer.
	//
	// This enforces that the final receive (for this paint cycle) on
	// gl.WorkAvailable happens before the send on endPaint.
	gl.Flush()
	w.endPaint <- paint.Event{} // TODO send real generation number
}
