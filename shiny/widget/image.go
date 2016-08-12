// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
	"image/draw"

	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
)

// TODO: mask and maskPoint, not just src and srcRect.

// TODO: be able to specify the draw operator: Src instead of Over.

// TODO: if the measured size differs from the actual size, specify a
// background color (or tile-able image like a checkerboard)? Specify a
// draw.Scaler from the golang.org/x/image/draw package? Be able to center the
// source image within the widget?

// Image is a leaf widget that paints an image.Image.
type Image struct {
	node.LeafEmbed
	Src     image.Image
	SrcRect image.Rectangle
}

// NewImage returns a new Image widget for the part of a source image defined
// by src and srcRect.
func NewImage(src image.Image, srcRect image.Rectangle) *Image {
	w := &Image{
		Src:     src,
		SrcRect: srcRect,
	}
	w.Wrapper = w
	return w
}

func (w *Image) Measure(t *theme.Theme, widthHint, heightHint int) {
	w.MeasuredSize = w.SrcRect.Size()
}

func (w *Image) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	if w.Src == nil {
		return nil
	}

	// wRect is the widget's layout rectangle, in dst's coordinate space.
	wRect := w.Rect.Add(origin)

	// sRect is the source image rectangle, in dst's coordinate space, so that
	// the upper-left corner of the source image rectangle aligns with the
	// upper-left corner of wRect.
	sRect := w.SrcRect.Add(wRect.Min.Sub(w.SrcRect.Min))

	draw.Draw(ctx.Dst, wRect.Intersect(sRect), w.Src, w.SrcRect.Min, draw.Over)
	return nil
}
