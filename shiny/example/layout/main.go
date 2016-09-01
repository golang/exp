// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build example
//
// This build tag means that "go install golang.org/x/exp/shiny/..." doesn't
// install this example program. Use "go run main.go" to run it or "go install
// -tags=example" to install it.

// layout is an example of a laying out a widget node tree. Real GUI programs
// won't need to do this explicitly, as the shiny/widget package will
// coordinate with the shiny/screen package to call Measure, Layout and Paint
// as necessary, and will re-layout widgets when windows are re-sized. This
// program merely demonstrates how a widget node tree can be rendered onto a
// statically sized RGBA image, for visual verification of widget code without
// having to bring up and manually inspect an interactive GUI window.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"

	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
)

var px = unit.Pixels

func colorPatch(c color.Color, w, h unit.Value) *widget.Sizer {
	return widget.NewSizer(w, h, widget.NewUniform(theme.StaticColor(c), nil))
}

func main() {
	t := theme.Default

	// Make the widget node tree.
	hf := widget.NewFlow(widget.AxisHorizontal,
		widget.NewLabel("Cyan:"),
		widget.WithLayoutData(
			colorPatch(color.RGBA{0x00, 0x7f, 0x7f, 0xff}, px(0), px(20)),
			widget.FlowLayoutData{AlongWeight: 1, ExpandAlong: true},
		),
		widget.NewLabel("Magenta:"),
		widget.WithLayoutData(
			colorPatch(color.RGBA{0x7f, 0x00, 0x7f, 0xff}, px(0), px(30)),
			widget.FlowLayoutData{AlongWeight: 2, ExpandAlong: true},
		),
		widget.NewLabel("Yellow:"),
		widget.WithLayoutData(
			colorPatch(color.RGBA{0x7f, 0x7f, 0x00, 0xff}, px(0), px(40)),
			widget.FlowLayoutData{AlongWeight: 3, ExpandAlong: true},
		),
	)

	vf := widget.NewFlow(widget.AxisVertical,
		colorPatch(color.RGBA{0xff, 0x00, 0x00, 0xff}, px(80), px(40)),
		colorPatch(color.RGBA{0x00, 0xff, 0x00, 0xff}, px(50), px(50)),
		colorPatch(color.RGBA{0x00, 0x00, 0xff, 0xff}, px(20), px(60)),
		widget.WithLayoutData(
			hf,
			widget.FlowLayoutData{ExpandAcross: true},
		),
		widget.NewLabel(fmt.Sprintf(
			"The black rectangle is 1.5 inches x 1 inch when viewed at %v DPI.", t.GetDPI())),
		widget.NewPadder(widget.AxisBoth, unit.Pixels(8),
			colorPatch(color.Black, unit.Inches(1.5), unit.Inches(1)),
		),
	)

	// Make the RGBA image.
	const width, height = 640, 480
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), t.GetPalette().Neutral(), image.Point{}, draw.Src)

	// Measure, layout and paint.
	vf.Measure(t, width, height)
	vf.Rect = rgba.Bounds()
	vf.Layout(t)
	vf.PaintBase(&node.PaintBaseContext{
		Theme: t,
		Dst:   rgba,
	}, image.Point{})

	// Encode to PNG.
	out, err := os.Create("out.png")
	if err != nil {
		log.Fatalf("os.Create: %v", err)
	}
	defer out.Close()
	if err := png.Encode(out, rgba); err != nil {
		log.Fatalf("png.Encode: %v", err)
	}
	fmt.Println("Wrote out.png OK.")
}
