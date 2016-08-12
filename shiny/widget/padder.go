// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
)

// Padder is a shell widget that adds a margin to the inner widget's measured
// size. That margin may be added horizontally (left and right), vertically
// (top and bottom) or both, determined by the Padder's axis.
//
// That marginal space is not considered part of the inner widget's geometry.
// For example, to make that space 'clickable', construct the Padder inside of
// an event handling widget instead of vice versa.
type Padder struct {
	node.ShellEmbed
	Axis   Axis
	Margin unit.Value
}

// NewPadder returns a new Padder widget.
func NewPadder(a Axis, margin unit.Value, inner node.Node) *Padder {
	w := &Padder{
		Axis:   a,
		Margin: margin,
	}
	w.Wrapper = w
	if inner != nil {
		w.Insert(inner, nil)
	}
	return w
}

func (w *Padder) Measure(t *theme.Theme, widthHint, heightHint int) {
	margin2 := t.Pixels(w.Margin).Round() * 2
	if w.Axis.Horizontal() && widthHint >= 0 {
		widthHint -= margin2
		if widthHint < 0 {
			widthHint = 0
		}
	}
	if w.Axis.Vertical() && heightHint >= 0 {
		heightHint -= margin2
		if heightHint < 0 {
			heightHint = 0
		}
	}
	w.ShellEmbed.Measure(t, widthHint, heightHint)
	if w.Axis.Horizontal() {
		w.MeasuredSize.X += margin2
	}
	if w.Axis.Vertical() {
		w.MeasuredSize.Y += margin2
	}
}

func (w *Padder) Layout(t *theme.Theme) {
	if c := w.FirstChild; c != nil {
		r := w.Rect.Sub(w.Rect.Min)
		inset := r.Inset(t.Pixels(w.Margin).Round())
		if w.Axis.Horizontal() {
			r.Min.X = inset.Min.X
			r.Max.X = inset.Max.X
		}
		if w.Axis.Vertical() {
			r.Min.Y = inset.Min.Y
			r.Max.Y = inset.Max.Y
		}
		c.Rect = r
		c.Wrapper.Layout(t)
	}
}
