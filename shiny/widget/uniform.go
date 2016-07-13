// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
)

// Uniform is a leaf widget that paints a uniform color, analogous to an
// image.Uniform.
type Uniform struct {
	node.LeafEmbed
	Uniform image.Uniform
}

// NewUniform returns a new Uniform widget of the given color.
func NewUniform(c color.Color) *Uniform {
	w := &Uniform{
		Uniform: image.Uniform{c},
	}
	w.Wrapper = w
	return w
}

func (w *Uniform) Paint(t *theme.Theme, dst *image.RGBA, origin image.Point) {
	w.Marks.UnmarkNeedsPaint()
	if w.Uniform.C == nil {
		return
	}
	draw.Draw(dst, w.Rect.Add(origin), &w.Uniform, image.Point{}, draw.Src)
}
