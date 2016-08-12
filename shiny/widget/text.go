// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
	"image/draw"

	"golang.org/x/exp/shiny/text"
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Text is a leaf widget that holds a text label.
type Text struct {
	node.LeafEmbed
	frame   text.Frame
	faceSet bool

	// TODO: scrolling, although should that be the responsibility of this
	// widget, the parent widget or something else?
}

// NewText returns a new Text widget.
func NewText(text string) *Text {
	w := &Text{}
	w.Wrapper = w
	if text != "" {
		c := w.frame.NewCaret()
		c.WriteString(text)
		c.Close()
	}
	return w
}

func (w *Text) setFace(t *theme.Theme) {
	// TODO: can a theme change at runtime, or can it be set only once, at
	// start-up?
	if !w.faceSet {
		w.faceSet = true
		// TODO: when is face released? Should we just unconditionally call
		// SetFace for every Measure, Layout and Paint? How do we avoid
		// excessive re-calculation of soft returns when re-using the same
		// logical face (as in "Times New Roman 12pt") even if using different
		// physical font.Face values (as each Face may have its own caches)?
		face := t.AcquireFontFace(theme.FontFaceOptions{})
		w.frame.SetFace(face)
	}
}

// TODO: should padding (and/or margin and border) be a universal concept and
// part of the node.Embed type instead of having each widget implement its own?

func (w *Text) padding(t *theme.Theme) int {
	return t.Pixels(unit.Ems(0.5)).Ceil()
}

func (w *Text) Measure(t *theme.Theme, widthHint, heightHint int) {
	w.setFace(t)
	padding := w.padding(t)

	if widthHint < 0 {
		w.frame.SetMaxWidth(0)
		w.MeasuredSize = image.Point{
			0, // TODO: this isn't right.
			w.frame.Height() + 2*padding,
		}
		return
	}

	maxWidth := fixed.I(widthHint - 2*padding)
	if maxWidth <= 1 {
		maxWidth = 1
	}
	w.frame.SetMaxWidth(maxWidth)

	w.MeasuredSize = image.Point{
		widthHint,
		w.frame.Height() + 2*padding,
	}
}

func (w *Text) Layout(t *theme.Theme) {
	w.setFace(t)
	padding := w.padding(t)
	maxWidth := fixed.I(w.Rect.Dx() - 2*padding)
	if maxWidth <= 1 {
		maxWidth = 1
	}
	w.frame.SetMaxWidth(maxWidth)
}

func (w *Text) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	dst := ctx.Dst.SubImage(w.Rect.Add(origin)).(*image.RGBA)
	if dst.Bounds().Empty() {
		return nil
	}

	face := ctx.Theme.AcquireFontFace(theme.FontFaceOptions{})
	defer ctx.Theme.ReleaseFontFace(theme.FontFaceOptions{}, face)
	m := face.Metrics()
	ascent := m.Ascent.Ceil()
	descent := m.Descent.Ceil()
	height := m.Height.Ceil()

	padding := w.padding(ctx.Theme)

	draw.Draw(dst, dst.Bounds(), ctx.Theme.GetPalette().Background(), image.Point{}, draw.Src)

	minDotY := fixed.I(dst.Bounds().Min.Y - descent)
	maxDotY := fixed.I(dst.Bounds().Max.Y + ascent)

	x0 := fixed.I(origin.X + w.Rect.Min.X + padding)
	d := font.Drawer{
		Dst:  dst,
		Src:  ctx.Theme.GetPalette().Foreground(),
		Face: face,
		Dot: fixed.Point26_6{
			X: x0,
			Y: fixed.I(origin.Y + w.Rect.Min.Y + padding + ascent),
		},
	}
	f := &w.frame
	for p := f.FirstParagraph(); p != nil; p = p.Next(f) {
		for l := p.FirstLine(f); l != nil; l = l.Next(f) {
			if d.Dot.Y > minDotY {
				if d.Dot.Y >= maxDotY {
					return nil
				}
				for b := l.FirstBox(f); b != nil; b = b.Next(f) {
					d.DrawBytes(b.TrimmedText(f))
					// TODO: adjust d.Dot.X for any ligatures?
				}
				d.Dot.X = x0
			}
			d.Dot.Y += fixed.I(height)
		}
	}
	return nil
}

func (w *Text) Paint(ctx *node.PaintContext, origin image.Point) error {
	// TODO: draw an optional border, whose color depends on whether w has the
	// keyboard focus.
	return w.LeafEmbed.Paint(ctx, origin)
}
