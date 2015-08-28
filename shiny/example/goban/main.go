// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore
//
// This build tag means that "go install golang.org/x/exp/shiny/..." doesn't
// install this example program. Use "go run main.go board.go xy.go" to run it.

// Goban is a simple example of a graphics program using shiny.
// It implements a Go board that two people can use to play the game.
// TODO: Improve the main function.
// TODO: Provide more functionality.
package main

import (
	"flag"
	"image"
	"image/color"
	stdDraw "image/draw"
	"log"
	"math/rand"
	"time"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

var dirty bool
var uploading bool

var scale = flag.Int("scale", 35, "`percent` to scale images (TODO: a poor design)")

func main() {
	flag.Parse()

	rand.Seed(int64(time.Now().Nanosecond()))
	board := NewBoard(9, *scale)

	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(nil)
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		r := board.image.Bounds()
		winSize := r.Size()
		var b screen.Buffer
		defer func() {
			if b != nil {
				b.Release()
			}
		}()

		var sz size.Event

		for e := range w.Events() {
			switch e := e.(type) {
			default:

			case mouse.Event:
				if e.Direction == mouse.DirRelease && e.Button != 0 {
					// Invert y. TODO: for Darwin gldriver bug that will be fixed by https://go-review.googlesource.com/#/c/13917/
					y := b.RGBA().Bounds().Dy() - int(e.Y)
					board.click(b.RGBA(), int(e.X), y, int(e.Button))
					dirty = true
				}

			case key.Event:
				if e.Code == key.CodeEscape {
					return
				}

			case paint.Event:
				// TODO: This check should save CPU time but causes flicker on Darwin.
				//if dirty && !uploading {
				w.Fill(sz.Bounds(), color.RGBA{0x00, 0x00, 0x3f, 0xff}, stdDraw.Src)
				w.Upload(image.Point{0, 0}, b, b.Bounds(), w) // TODO: On Darwin always writes to 0,0, ignoring first arg.
				dirty = false
				uploading = true
				//}
				w.EndPaint(e)

			case screen.UploadedEvent:
				// No-op.
				uploading = false

			case size.Event:
				// TODO: Set board size.
				sz = e
				if b != nil {
					b.Release()
				}
				winSize = image.Point{sz.WidthPx, sz.HeightPx}
				b, err = s.NewBuffer(winSize)
				if err != nil {
					log.Fatal(err)
				}
				render(b.RGBA(), board)

			case error:
				log.Print(e)
			}
		}
	})
}

func render(m *image.RGBA, board *Board) {
	board.Draw(m)
	dirty = true
}
