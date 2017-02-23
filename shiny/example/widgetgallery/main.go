// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build example
//
// This build tag means that "go install golang.org/x/exp/shiny/..." doesn't
// install this example program. Use "go run main.go" to run it or "go install
// -tags=example" to install it.

// Widgetgallery exhibits the shiny/widget package's widget set.
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
		w.Mark(node.MarkNeedsPaintBase)
	}
	return node.Handled
}

func (w *custom) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	draw.Draw(ctx.Dst, w.Rect.Add(origin), uniforms[w.index], image.Point{}, draw.Src)
	return nil
}

func main() {
	log.SetFlags(0)
	driver.Main(func(s screen.Screen) {
		// TODO: create a bunch of standard widgets: buttons, labels, etc.
		w := widget.NewSheet(newCustom())
		if err := widget.RunWindow(s, w, &widget.RunWindowOptions{
			NewWindowOptions: screen.NewWindowOptions{
				Title: "WidgetGallery Shiny Example",
			},
		}); err != nil {
			log.Fatal(err)
		}
	})
}
