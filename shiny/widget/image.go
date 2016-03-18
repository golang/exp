// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
)

// Image is a widget that holds an image.Image.
type Image struct{ *Node }

// NewImage returns a new Image widget.
func NewImage() Image { return Image{&Node{Class: ImageClass{}}} }

func (o Image) Image() image.Image     { v, _ := o.ClassData.(image.Image); return v }
func (o Image) SetImage(v image.Image) { o.ClassData = v }

// ImageClass is the Class for Image nodes.
type ImageClass struct{ LeafClassEmbed }

func (k ImageClass) Measure(n *Node) image.Point {
	o := Image{n}
	if m := o.Image(); m != nil {
		return m.Bounds().Size()
	}
	return image.Point{}
}

func (k ImageClass) Paint(n *Node, dst *image.RGBA) {
	o := Image{n}
	if m := o.Image(); m != nil {
		// TODO: copy m to dst with the appropriate offset and clip.
	}
}
