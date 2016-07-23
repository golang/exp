// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build example
//
// This build tag means that "go install golang.org/x/exp/shiny/..." doesn't
// install this example program. Use "go run main.go" to run it or "go install
// -tags=example" to install it.

// Gallery demonstrates the shiny/widget package.
package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/gesture"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/widget"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
)

var uniforms = [...]*image.Uniform{
	image.NewUniform(color.RGBA{0xbf, 0x00, 0x00, 0xff}),
	image.NewUniform(color.RGBA{0x9f, 0x9f, 0x00, 0xff}),
	image.NewUniform(color.RGBA{0x00, 0xbf, 0x00, 0xff}),
	image.NewUniform(color.RGBA{0x00, 0x9f, 0x9f, 0xff}),
	image.NewUniform(color.RGBA{0x00, 0x00, 0xbf, 0xff}),
	image.NewUniform(color.RGBA{0x9f, 0x00, 0x9f, 0xff}),
}

// custom is a custom widget.
type custom struct {
	node.LeafEmbed
	index int
}

func newCustom() *custom {
	w := &custom{}
	w.Wrapper = w
	return w
}

func (w *custom) OnInputEvent(e interface{}, origin image.Point) node.EventHandled {
	switch e := e.(type) {
	case gesture.Event:
		if e.Type != gesture.TypeTap {
			break
		}
		w.index++
		if w.index == len(uniforms) {
			w.index = 0
		}
		w.Mark(node.MarkNeedsPaint)
	}
	return node.Handled
}

func (w *custom) Paint(t *theme.Theme, dst *image.RGBA, origin image.Point) {
	w.Marks.UnmarkNeedsPaint()
	draw.Draw(dst, w.Rect.Add(origin), uniforms[w.index], image.Point{}, draw.Src)
}

func main() {
	log.SetFlags(0)
	driver.Main(func(s screen.Screen) {
		// TODO: create a bunch of standard widgets: buttons, labels, etc.
		w := newCustom()
		if err := widget.RunWindow(s, w, nil); err != nil {
			log.Fatal(err)
		}
	})
}
