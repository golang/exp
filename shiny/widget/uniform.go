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

func (w *Uniform) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	if w.ThemeColor != nil {
		src := w.ThemeColor.Uniform(ctx.Theme)
		// TODO: should draw.Src be draw.Over?
		draw.Draw(ctx.Dst, w.Rect.Add(origin), src, image.Point{}, draw.Src)
	}
	if c := w.FirstChild; c != nil {
		return c.Wrapper.PaintBase(ctx, origin.Add(w.Rect.Min))
	}
	return nil
}
