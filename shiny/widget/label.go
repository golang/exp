// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Label is a leaf widget that holds a text label.
type Label struct{ *Node }

// NewLabel returns a new Label widget.
//
// TODO: take a "text string" argument?
func NewLabel() Label { return Label{&Node{Class: LabelClass{}}} }

func (o Label) Text() string     { v, _ := o.ClassData.(string); return v }
func (o Label) SetText(v string) { o.ClassData = v }

// LabelClass is the Class for Label nodes.
type LabelClass struct{ LeafClassEmbed }

func (k LabelClass) Measure(n *Node, t *Theme) {
	o := Label{n}

	f := t.AcquireFontFace(FontFaceOptions{})
	defer t.ReleaseFontFace(FontFaceOptions{}, f)
	m := f.Metrics()

	n.MeasuredSize.X = font.MeasureString(f, o.Text()).Ceil()
	n.MeasuredSize.Y = m.Ascent.Ceil() + m.Descent.Ceil()
}

func (k LabelClass) Paint(n *Node, t *Theme, dst *image.RGBA, origin image.Point) {
	o := Label{n}

	f := t.AcquireFontFace(FontFaceOptions{})
	defer t.ReleaseFontFace(FontFaceOptions{}, f)
	m := f.Metrics()

	d := font.Drawer{
		Dst:  dst,
		Src:  t.GetPalette().Foreground,
		Face: f,
		Dot: fixed.Point26_6{
			X: fixed.I(origin.X + n.Rect.Min.X),
			Y: fixed.I(origin.Y + n.Rect.Min.Y + m.Ascent.Ceil()),
		},
	}
	d.DrawString(o.Text())
}
