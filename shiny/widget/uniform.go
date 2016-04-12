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
			Class: UniformClass{},
			ClassData: &uniformClassData{
				u: image.NewUniform(c),
				w: naturalWidth,
				h: naturalHeight,
			},
		},
	}
}

func (o Uniform) Color() color.Color            { return o.classData().u.C }
func (o Uniform) SetColor(v color.Color)        { o.classData().u.C = v }
func (o Uniform) NaturalWidth() unit.Value      { return o.classData().w }
func (o Uniform) SetNaturalWidth(v unit.Value)  { o.classData().w = v }
func (o Uniform) NaturalHeight() unit.Value     { return o.classData().h }
func (o Uniform) SetNaturalHeight(v unit.Value) { o.classData().h = v }

func (o Uniform) classData() *uniformClassData { return o.ClassData.(*uniformClassData) }

type uniformClassData struct {
	u *image.Uniform
	w unit.Value
	h unit.Value
}

// UniformClass is the Class for Uniform nodes.
type UniformClass struct{ LeafClassEmbed }

func (k UniformClass) Measure(n *Node, t *Theme) {
	d := Uniform{n}.classData()
	n.MeasuredSize.X = t.Pixels(d.w).Round()
	n.MeasuredSize.Y = t.Pixels(d.h).Round()
}

func (k UniformClass) Paint(n *Node, t *Theme, dst *image.RGBA, origin image.Point) {
	d := Uniform{n}.classData()
	draw.Draw(dst, n.Rect.Add(origin), d.u, image.Point{}, draw.Src)
}
