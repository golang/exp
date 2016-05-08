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

// TODO: be able to override the natural width and height, e.g. to specify a
// button's image in inches instead of (DPI-independent) pixels? Should that be
// the responsibility of this widget (Image) or of a Sizer shell widget?

// TODO: if the measured size differs from the actual size, specify a
// background color (or tile-able image like a checkerboard)? Specify a
// draw.Scaler from the golang.org/x/image/draw package? Be able to center the
// source image within the widget?

// Image is a leaf widget that paints an image.Image.
type Image struct{ *node.Node }

// NewImage returns a new Image widget for the part of a source image defined
// by src and srcRect.
func NewImage(src image.Image, srcRect image.Rectangle) Image {
	return Image{
		&node.Node{
			Class: &imageClass{
				src:     src,
				srcRect: srcRect,
			},
		},
	}
}

func (o Image) Src() image.Image             { return o.Class.(*imageClass).src }
func (o Image) SetSrc(v image.Image)         { o.Class.(*imageClass).src = v }
func (o Image) SrcRect() image.Rectangle     { return o.Class.(*imageClass).srcRect }
func (o Image) SetSrcRect(v image.Rectangle) { o.Class.(*imageClass).srcRect = v }

type imageClass struct {
	node.LeafClassEmbed
	src     image.Image
	srcRect image.Rectangle
}

func (k *imageClass) Measure(n *node.Node, t *theme.Theme) {
	n.MeasuredSize = k.srcRect.Size()
}

func (k *imageClass) Paint(n *node.Node, t *theme.Theme, dst *image.RGBA, origin image.Point) {
	if k.src == nil {
		return
	}

	// nRect is the node's layout rectangle, in dst's coordinate space.
	nRect := n.Rect.Add(origin)

	// sRect is the source image rectangle, in dst's coordinate space, so that
	// the upper-left corner of the source image rectangle aligns with the
	// upper-left corner of nRect.
	sRect := k.srcRect.Add(nRect.Min.Sub(k.srcRect.Min))

	draw.Draw(dst, nRect.Intersect(sRect), k.src, k.srcRect.Min, draw.Over)
}
