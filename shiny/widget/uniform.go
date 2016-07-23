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

// Uniform is a shell widget that paints a uniform color, analogous to an
// image.Uniform.
type Uniform struct {
	node.ShellEmbed
	ThemeColor theme.Color
}

// NewUniform returns a new Uniform widget of the given color.
func NewUniform(c theme.Color, inner node.Node) *Uniform {
	w := &Uniform{
		ThemeColor: c,
	}
	w.Wrapper = w
	if inner != nil {
		w.Insert(inner, nil)
	}
	return w
}

func (w *Uniform) Paint(t *theme.Theme, dst *image.RGBA, origin image.Point) {
	w.Marks.UnmarkNeedsPaint()
	if w.ThemeColor != nil {
		// TODO: should draw.Src be draw.Over?
		draw.Draw(dst, w.Rect.Add(origin), w.ThemeColor.Uniform(t), image.Point{}, draw.Src)
	}
	if c := w.FirstChild; c != nil {
		c.Wrapper.Paint(t, dst, origin.Add(w.Rect.Min))
	}
}
