// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
)

// Sizer is a shell widget that overrides its child's measured size.
type Sizer struct {
	node.ShellEmbed
	NaturalWidth  unit.Value
	NaturalHeight unit.Value
}

// NewSizer returns a new Sizer widget of the given natural size. Its parent
// widget may lay it out at a different size than its natural size, such as
// expanding to fill a panel's width.
func NewSizer(naturalWidth, naturalHeight unit.Value, inner node.Node) *Sizer {
	w := &Sizer{
		NaturalWidth:  naturalWidth,
		NaturalHeight: naturalHeight,
	}
	w.Wrapper = w
	if inner != nil {
		w.Insert(inner, nil)
	}
	return w
}

func (w *Sizer) Measure(t *theme.Theme, widthHint, heightHint int) {
	w.MeasuredSize.X = t.Pixels(w.NaturalWidth).Round()
	w.MeasuredSize.Y = t.Pixels(w.NaturalHeight).Round()
	if c := w.FirstChild; c != nil {
		c.Wrapper.Measure(t, w.MeasuredSize.X, w.MeasuredSize.Y)
	}
}
