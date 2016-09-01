// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"

	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
)

// TODO: padding, alignment.

// Flow is a container widget that lays out its children in sequence along an
// axis, either horizontally or vertically. The children's laid out size may
// differ from their natural size, along or across that axis, if a child's
// LayoutData is a FlowLayoutData.
type Flow struct {
	node.ContainerEmbed
	Axis Axis
}

// NewFlow returns a new Flow widget containing the given children.
func NewFlow(a Axis, children ...node.Node) *Flow {
	w := &Flow{
		Axis: a,
	}
	w.Wrapper = w
	for _, c := range children {
		w.Insert(c, nil)
	}
	return w
}

func (w *Flow) Measure(t *theme.Theme, widthHint, heightHint int) {
	if w.Axis != AxisHorizontal && w.Axis != AxisVertical {
		w.ContainerEmbed.Measure(t, widthHint, heightHint)
		return
	}

	if w.Axis == AxisHorizontal {
		widthHint = node.NoHint
	}
	if w.Axis == AxisVertical {
		heightHint = node.NoHint
	}

	mSize := image.Point{}
	for c := w.FirstChild; c != nil; c = c.NextSibling {
		c.Wrapper.Measure(t, widthHint, heightHint)
		if w.Axis == AxisHorizontal {
			mSize.X += c.MeasuredSize.X
			if mSize.Y < c.MeasuredSize.Y {
				mSize.Y = c.MeasuredSize.Y
			}
		} else {
			mSize.Y += c.MeasuredSize.Y
			if mSize.X < c.MeasuredSize.X {
				mSize.X = c.MeasuredSize.X
			}
		}
	}
	w.MeasuredSize = mSize
}

func (w *Flow) Layout(t *theme.Theme) {
	if w.Axis != AxisHorizontal && w.Axis != AxisVertical {
		w.ContainerEmbed.Layout(t)
		return
	}

	extra, totalExpandWeight, totalShrinkWeight := 0, 0, 0
	if w.Axis == AxisHorizontal {
		extra = w.Rect.Dx()
	} else {
		extra = w.Rect.Dy()
	}
	for c := w.FirstChild; c != nil; c = c.NextSibling {
		if d, ok := c.LayoutData.(FlowLayoutData); ok && d.AlongWeight > 0 {
			if d.AlongWeight <= 0 {
				continue
			}
			if d.ExpandAlong {
				totalExpandWeight += d.AlongWeight
			}
			if d.ShrinkAlong {
				totalShrinkWeight += d.AlongWeight
			}
		}
		if w.Axis == AxisHorizontal {
			extra -= c.MeasuredSize.X
		} else {
			extra -= c.MeasuredSize.Y
		}
	}
	expand, shrink, totalWeight := extra > 0, extra < 0, 0
	if expand {
		if totalExpandWeight == 0 {
			expand = false
		} else {
			totalWeight = totalExpandWeight
		}
	}
	if shrink {
		if totalShrinkWeight == 0 {
			shrink = false
		} else {
			totalWeight = totalShrinkWeight
		}
	}

	p := image.Point{}
	for c := w.FirstChild; c != nil; c = c.NextSibling {
		q := p.Add(c.MeasuredSize)
		if d, ok := c.LayoutData.(FlowLayoutData); ok {
			if d.AlongWeight > 0 {
				if (expand && d.ExpandAlong) || (shrink && d.ShrinkAlong) {
					delta := extra * d.AlongWeight / totalWeight
					extra -= delta
					totalWeight -= d.AlongWeight
					if w.Axis == AxisHorizontal {
						q.X += delta
						if q.X < p.X {
							q.X = p.X
						}
					} else {
						q.Y += delta
						if q.Y < p.Y {
							q.Y = p.Y
						}
					}
				}
			}

			if w.Axis == AxisHorizontal {
				q.Y = stretchAcross(q.Y, w.Rect.Dy(), d.ExpandAcross, d.ShrinkAcross)
			} else {
				q.X = stretchAcross(q.X, w.Rect.Dx(), d.ExpandAcross, d.ShrinkAcross)
			}
		}
		c.Rect = image.Rectangle{
			Min: p,
			Max: q,
		}
		c.Wrapper.Layout(t)
		if w.Axis == AxisHorizontal {
			p.X = q.X
		} else {
			p.Y = q.Y
		}
	}
}

func stretchAcross(child, parent int, expand, shrink bool) int {
	if (expand && child < parent) || (shrink && child > parent) {
		return parent
	}
	return child
}

// FlowLayoutData is the node LayoutData type for a Flow's children.
type FlowLayoutData struct {
	// AlongWeight is the relative weight for distributing any space surplus or
	// deficit along the Flow's axis. For example, if an AxisHorizontal Flow's
	// Rect width was 100 pixels greater than the sum of its children's natural
	// widths, and three children had non-zero FlowLayoutData.AlongWeight
	// values 6, 3 and 1 (and their FlowLayoutData.ExpandAlong values were
	// true) then those children's laid out widths would be larger than their
	// natural widths by 60, 30 and 10 pixels.
	//
	// A negative AlongWeight is equivalent to zero.
	AlongWeight int

	// ExpandAlong is whether the child's laid out size should increase along
	// the Flow's axis, based on AlongWeight, if there is a space surplus (the
	// children's measured size total less than the parent's size). To allow
	// size decreases as well as increases, set ShrinkAlong.
	ExpandAlong bool

	// ShrinkAlong is whether the child's laid out size should decrease along
	// the Flow's axis, based on AlongWeight, if there is a space deficit (the
	// children's measured size total more than the parent's size). To allow
	// size increases as well as decreases, set ExpandAlong.
	ShrinkAlong bool

	// ExpandAcross is whether the child's laid out size should increase along
	// the Flow's cross-axis if there is a space surplus (the child's measured
	// size is less than the parent's size). To allow size decreases as well as
	// increases, set ShrinkAcross.
	//
	// For example, if an AxisHorizontal Flow's Rect height was 80 pixels, any
	// child whose FlowLayoutData.ExpandAcross was true would also be laid out
	// with at least an 80 pixel height.
	ExpandAcross bool

	// ShrinkAcross is whether the child's laid out size should decrease along
	// the Flow's cross-axis if there is a space deficit (the child's measured
	// size is more than the parent's size). To allow size increases as well as
	// decreases, set ExpandAcross.
	//
	// For example, if an AxisHorizontal Flow's Rect height was 80 pixels, any
	// child whose FlowLayoutData.ShrinkAcross was true would also be laid out
	// with at most an 80 pixel height.
	ShrinkAcross bool
}
