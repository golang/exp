// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flex

import (
	"image"
	"image/color"
	"testing"

	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget"
	"golang.org/x/exp/shiny/widget/node"
)

type layoutTest struct {
	direction    Direction
	wrap         FlexWrap
	alignContent AlignContent
	size         image.Point       // size of container
	measured     [][2]float64      // MeasuredSize of child elements
	layoutData   []LayoutData      // LayoutData of child elements
	want         []image.Rectangle // final Rect of child elements
}

var colors = []color.RGBA{
	{0x00, 0x7f, 0x7f, 0xff}, // Cyan
	{0x7f, 0x00, 0x7f, 0xff}, // Magenta
	{0x7f, 0x7f, 0x00, 0xff}, // Yellow
	{0xff, 0x00, 0x00, 0xff}, // Red
	{0x00, 0xff, 0x00, 0xff}, // Green
	{0x00, 0x00, 0xff, 0xff}, // Blue
}

var layoutTests = []layoutTest{{
	size:     image.Point{100, 100},
	measured: [][2]float64{{100, 100}},
	want: []image.Rectangle{
		image.Rect(0, 0, 100, 100),
	},
}}

func TestLayout(t *testing.T) {
	for testNum, test := range layoutTests {
		w := NewFlex()
		w.Direction = test.direction
		w.Wrap = test.wrap
		w.AlignContent = test.alignContent

		var children []node.Node
		for i, sz := range test.measured {
			n := widget.NewUniform(colors[i], unit.Pixels(sz[0]), unit.Pixels(sz[1]))
			if test.layoutData != nil {
				n.LayoutData = test.layoutData[i]
			}
			w.AppendChild(n)
			children = append(children, n)
		}

		w.Measure(nil)
		w.Rect = image.Rectangle{Max: test.size}
		w.Layout(nil)

		bad := false
		for i, n := range children {
			if n.Wrappee().Rect != test.want[i] {
				bad = true
				break
			}
		}
		if bad {
			t.Logf("Bad testNum %d", testNum)
			// TODO print html so we can see the correct layout
		}
		for i, n := range children {
			if got, want := n.Wrappee().Rect, test.want[i]; got != want {
				t.Errorf("[%d].Rect=%v, want %v", i, got, want)
			}
		}
	}
}
