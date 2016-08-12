// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"

	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Label is a leaf widget that holds a text label.
type Label struct {
	node.LeafEmbed
	Text       string
	ThemeColor theme.Color
}

// NewLabel returns a new Label widget.
func NewLabel(text string) *Label {
	w := &Label{
		Text: text,
	}
	w.Wrapper = w
	return w
}

func (w *Label) Measure(t *theme.Theme, widthHint, heightHint int) {
	face := t.AcquireFontFace(theme.FontFaceOptions{})
	defer t.ReleaseFontFace(theme.FontFaceOptions{}, face)
	m := face.Metrics()

	// TODO: padding, to match a Text widget?

	w.MeasuredSize.X = font.MeasureString(face, w.Text).Ceil()
	w.MeasuredSize.Y = m.Ascent.Ceil() + m.Descent.Ceil()
}

func (w *Label) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	dst := ctx.Dst.SubImage(w.Rect.Add(origin)).(*image.RGBA)
	if dst.Bounds().Empty() {
		return nil
	}

	face := ctx.Theme.AcquireFontFace(theme.FontFaceOptions{})
	defer ctx.Theme.ReleaseFontFace(theme.FontFaceOptions{}, face)
	m := face.Metrics()
	ascent := m.Ascent.Ceil()

	tc := w.ThemeColor
	if tc == nil {
		tc = theme.Foreground
	}

	d := font.Drawer{
		Dst:  dst,
		Src:  tc.Uniform(ctx.Theme),
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(origin.X + w.Rect.Min.X),
			Y: fixed.I(origin.Y + w.Rect.Min.Y + ascent),
		},
	}
	d.DrawString(w.Text)
	return nil
}
