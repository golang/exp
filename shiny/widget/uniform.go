// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/exp/shiny/unit"
)

// Uniform is a leaf widget that paints a uniform color, analogous to an
// image.Uniform.
type Uniform struct{ *Node }

// NewUniform returns a new Uniform widget of the given color and natural size.
// Its parent widget may lay it out at a different size than its natural size,
// such as expanding to fill a panel's width.
func NewUniform(c color.Color, naturalWidth, naturalHeight unit.Value) Uniform {
	return Uniform{
		&Node{
			Class: &uniformClass{
				u: image.NewUniform(c),
				w: naturalWidth,
				h: naturalHeight,
			},
		},
	}
}

func (o Uniform) Color() color.Color            { return o.Class.(*uniformClass).u.C }
func (o Uniform) SetColor(v color.Color)        { o.Class.(*uniformClass).u.C = v }
func (o Uniform) NaturalWidth() unit.Value      { return o.Class.(*uniformClass).w }
func (o Uniform) SetNaturalWidth(v unit.Value)  { o.Class.(*uniformClass).w = v }
func (o Uniform) NaturalHeight() unit.Value     { return o.Class.(*uniformClass).h }
func (o Uniform) SetNaturalHeight(v unit.Value) { o.Class.(*uniformClass).h = v }

type uniformClass struct {
	LeafClassEmbed
	u *image.Uniform
	w unit.Value
	h unit.Value
}

func (k *uniformClass) Measure(n *Node, t *Theme) {
	n.MeasuredSize.X = t.Pixels(k.w).Round()
	n.MeasuredSize.Y = t.Pixels(k.h).Round()
}

func (k *uniformClass) Paint(n *Node, t *Theme, dst *image.RGBA, origin image.Point) {
	draw.Draw(dst, n.Rect.Add(origin), k.u, image.Point{}, draw.Src)
}
