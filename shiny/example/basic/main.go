// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore
//
// This build tag means that "go install golang.org/x/exp/shiny/..." doesn't
// install this example program. Use "go run main.go" to run it.

// Basic is a basic example of a graphical application.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

var (
	blue0 = color.RGBA{0x00, 0x00, 0x1f, 0xff}
	blue1 = color.RGBA{0x00, 0x00, 0x3f, 0xff}
)

func main() {
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(nil)
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		winSize := image.Point{256, 256}
		b, err := s.NewBuffer(winSize)
		if err != nil {
			log.Fatal(err)
		}
		defer b.Release()
		drawGradient(b.RGBA())

		t, err := s.NewTexture(winSize)
		if err != nil {
			log.Fatal(err)
		}
		defer t.Release()
		t.Upload(image.Point{}, b, b.Bounds())

		var sz size.Event
		for e := range w.Events() {
			switch e := e.(type) {
			default:
				// TODO: be more interesting.
				fmt.Printf("got %#v\n", e)

			case key.Event:
				fmt.Printf("got %v\n", e)
				if e.Code == key.CodeEscape {
					return
				}

			case paint.Event:
				w.Fill(sz.Bounds(), blue0, draw.Src)
				w.Fill(sz.Bounds().Inset(10), blue1, draw.Src)
				w.Upload(image.Point{}, b, b.Bounds())
				c := math.Cos(math.Pi / 6)
				s := math.Sin(math.Pi / 6)
				w.Draw(f64.Aff3{
					+c, -s, 100,
					+s, +c, 200,
				}, t, t.Bounds(), screen.Over, nil)
				w.Publish()

			case size.Event:
				sz = e

			case error:
				log.Print(e)
			}
		}
	})
}

func drawGradient(m *image.RGBA) {
	b := m.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if x%64 == 0 || y%64 == 0 {
				m.SetRGBA(x, y, color.RGBA{0xff, 0xff, 0xff, 0xff})
			} else if x%64 == 63 || y%64 == 63 {
				m.SetRGBA(x, y, color.RGBA{0x00, 0x00, 0xff, 0xff})
			} else {
				m.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 0x00, 0xff})
			}
		}
	}
}
