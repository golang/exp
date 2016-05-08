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
type Label struct{ *node.Node }

// NewLabel returns a new Label widget.
func NewLabel(text string) Label {
	return Label{
		&node.Node{
			Class: &labelClass{
				text: text,
			},
		},
	}
}

func (o Label) Text() string     { return o.Class.(*labelClass).text }
func (o Label) SetText(v string) { o.Class.(*labelClass).text = v }

type labelClass struct {
	node.LeafClassEmbed
	text string
}

func (k *labelClass) Measure(n *node.Node, t *theme.Theme) {
	f := t.AcquireFontFace(theme.FontFaceOptions{})
	defer t.ReleaseFontFace(theme.FontFaceOptions{}, f)
	m := f.Metrics()

	n.MeasuredSize.X = font.MeasureString(f, k.text).Ceil()
	n.MeasuredSize.Y = m.Ascent.Ceil() + m.Descent.Ceil()
}

func (k *labelClass) Paint(n *node.Node, t *theme.Theme, dst *image.RGBA, origin image.Point) {
	f := t.AcquireFontFace(theme.FontFaceOptions{})
	defer t.ReleaseFontFace(theme.FontFaceOptions{}, f)
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
	d.DrawString(k.text)
}
