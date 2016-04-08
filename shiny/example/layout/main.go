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

	"golang.org/x/exp/shiny/widget"
)

func mkImage(width, height int, c color.RGBA) *widget.Node {
	src := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(src, src.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)

	m := widget.NewImage()
	m.SetImage(src)
	return m.Node
}

func main() {
	// Make the widget node tree.
	vf := widget.NewFlow()
	vf.SetAxis(widget.AxisVertical)
	vf.AppendChild(mkImage(80, 40, color.RGBA{0xff, 0x00, 0x00, 0xff}))
	vf.AppendChild(mkImage(50, 50, color.RGBA{0x00, 0xff, 0x00, 0xff}))
	vf.AppendChild(mkImage(20, 60, color.RGBA{0x00, 0x00, 0xff, 0xff}))

	// Make the RGBA image.
	t := widget.DefaultTheme
	rgba := image.NewRGBA(image.Rect(0, 0, 640, 480))
	draw.Draw(rgba, rgba.Bounds(), t.Palette().Neutral, image.Point{}, draw.Src)

	// Measure, layout and paint.
	vf.Class.Measure(vf.Node, t)
	vf.Rect = rgba.Bounds()
	vf.Class.Layout(vf.Node, t)
	vf.Class.Paint(vf.Node, t, rgba)

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
