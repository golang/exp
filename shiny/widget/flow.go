// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
)

// TODO: padding, alignment.

// Flow is a container widget that lays out its children in sequence along an
// axis, either horizontally or vertically.
type Flow struct{ *Node }

// NewFlow returns a new Flow widget.
func NewFlow(a Axis) Flow {
	return Flow{
		&Node{
			Class:     FlowClass{},
			ClassData: a,
		},
	}
}

func (o Flow) Axis() Axis     { v, _ := o.ClassData.(Axis); return v }
func (o Flow) SetAxis(v Axis) { o.ClassData = v }

// FlowClass is the Class for Flow nodes.
type FlowClass struct{ ContainerClassEmbed }

func (k FlowClass) Measure(n *Node, t *Theme) {
	o := Flow{n}
	axis := o.Axis()
	if axis != AxisHorizontal && axis != AxisVertical {
		k.ContainerClassEmbed.Measure(n, t)
		return
	}

	mSize := image.Point{}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		c.Measure(t)
		if axis == AxisHorizontal {
			if mSize.Y < c.MeasuredSize.Y {
				mSize.Y = c.MeasuredSize.Y
			}
		} else {
			if mSize.X < c.MeasuredSize.X {
				mSize.X = c.MeasuredSize.X
			}
		}
	}
	n.MeasuredSize = mSize
}

func (k FlowClass) Layout(n *Node, t *Theme) {
	o := Flow{n}
	axis := o.Axis()
	if axis != AxisHorizontal && axis != AxisVertical {
		k.ContainerClassEmbed.Layout(n, t)
		return
	}

	min := image.Point{}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		c.Rect = image.Rectangle{
			Min: min,
			Max: min.Add(c.MeasuredSize),
		}
		c.Layout(t)
		if axis == AxisHorizontal {
			min.X += c.MeasuredSize.X
		} else {
			min.Y += c.MeasuredSize.Y
		}
	}
}
