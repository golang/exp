// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore
//
// This build tag means that "go install golang.org/x/exp/shiny/..." doesn't
// install this example program. Use "go run main.go" to explicitly run it.

// Program basic is a basic example of a graphical application.
package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
)

func main() {
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(nil)
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		b, err := s.NewBuffer(image.Point{256, 256})
		if err != nil {
			log.Fatal(err)
		}
		defer b.Release()
		fill(b.RGBA())
		// TODO: w.Upload(etc, b)

		for e := range w.Events() {
			// TODO: be more interesting.
			fmt.Println(e)
		}
	})
}

func fill(m *image.RGBA) {
	b := m.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			m.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 0x00, 0xff})
		}
	}
}
