// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
	"image/draw"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/lifecycle"
)

// TODO: scrolling.

// Sheet is a shell widget that provides *image.RGBA pixel buffers (analogous
// to blank sheets of paper) for its descendent widgets to paint on, via their
// PaintBase methods. Such buffers may be cached and their contents re-used for
// multiple paints, which can make scrolling and animation smoother and more
// efficient.
//
// A simple app may have only one Sheet, near the root of its widget tree. A
// more complicated app may have multiple Sheets. For example, consider a text
// editor consisting of a small header bar and a large text widget. Those two
// nodes may be backed by two separate Sheets, since scrolling the latter
// should not scroll the former.
type Sheet struct {
	node.ShellEmbed
	buf screen.Buffer
	tex screen.Texture
}

// NewSheet returns a new Sheet widget.
func NewSheet(inner node.Node) *Sheet {
	w := &Sheet{}
	w.Wrapper = w
	if inner != nil {
		w.Insert(inner, nil)
	}
	return w
}

func (w *Sheet) release() {
	if w.buf != nil {
		w.buf.Release()
		w.buf = nil
	}
	if w.tex != nil {
		w.tex.Release()
		w.tex = nil
	}
}

func (w *Sheet) Paint(ctx *node.PaintContext, origin image.Point) (retErr error) {
	w.Marks.UnmarkNeedsPaint()
	c := w.FirstChild
	if c == nil {
		w.release()
		return nil
	}

	fresh, size := false, w.Rect.Size()
	if w.buf != nil && w.buf.Size() != size {
		w.release()
	}
	if w.buf == nil {
		w.buf, retErr = ctx.Screen.NewBuffer(size)
		if retErr != nil {
			w.release()
			return retErr
		}
		w.tex, retErr = ctx.Screen.NewTexture(size)
		if retErr != nil {
			w.release()
			return retErr
		}
		fresh = true
	}
	if fresh || c.Marks.NeedsPaintBase() {
		c.Wrapper.PaintBase(&node.PaintBaseContext{
			Theme: ctx.Theme,
			Dst:   w.buf.RGBA(),
		}, image.Point{})
	}

	w.tex.Upload(image.Point{}, w.buf, w.buf.Bounds())

	src2dst := ctx.Src2Dst
	translate(&src2dst,
		float64(origin.X+w.Rect.Min.X),
		float64(origin.Y+w.Rect.Min.Y),
	)
	// TODO: should draw.Over be configurable?
	ctx.Drawer.Draw(src2dst, w.tex, w.tex.Bounds(), draw.Over, nil)

	return c.Wrapper.Paint(ctx, origin.Add(w.Rect.Min))
}

func translate(a *f64.Aff3, tx, ty float64) {
	a[2] += a[0]*tx + a[1]*ty
	a[5] += a[3]*tx + a[4]*ty
}

func (w *Sheet) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	// Do not recursively call PaintBase on our children. We create our own
	// buffers, and Sheet.Paint will call PaintBase with our PaintBaseContext
	// instead of our ancestor's.
	return nil
}

func (w *Sheet) OnChildMarked(child node.Node, newMarks node.Marks) {
	if newMarks&node.MarkNeedsPaintBase != 0 {
		newMarks &^= node.MarkNeedsPaintBase
		newMarks |= node.MarkNeedsPaint
	}
	w.Mark(newMarks)
}

func (w *Sheet) OnLifecycleEvent(e lifecycle.Event) {
	if e.Crosses(lifecycle.StageVisible) == lifecycle.CrossOff {
		w.release()
	}
}
