// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
	"image/draw"
)

// TODO: have source-rect, mask, mask-point properties as per draw.Draw arguments.

// Image is a leaf widget that holds an image.Image.
type Image struct{ *Node }

// NewImage returns a new Image widget.
func NewImage(m image.Image) Image {
	return Image{
		&Node{
			Class:     ImageClass{},
			ClassData: m,
		},
	}
}

func (o Image) Image() image.Image     { v, _ := o.ClassData.(image.Image); return v }
func (o Image) SetImage(v image.Image) { o.ClassData = v }

// ImageClass is the Class for Image nodes.
type ImageClass struct{ LeafClassEmbed }

func (k ImageClass) Measure(n *Node, t *Theme) {
	o := Image{n}
	if m := o.Image(); m != nil {
		n.MeasuredSize = m.Bounds().Size()
	} else {
		n.MeasuredSize = image.Point{}
	}
}

func (k ImageClass) Paint(n *Node, t *Theme, dst *image.RGBA, origin image.Point) {
	o := Image{n}
	if m := o.Image(); m != nil {
		draw.Draw(dst, n.Rect.Add(origin), m, image.Point{}, draw.Src)
	}
}
