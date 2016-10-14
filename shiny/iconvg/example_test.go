// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg_test

import (
	"image"
	"image/draw"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/exp/shiny/iconvg"
)

func Example() {
	ivgData, err := ioutil.ReadFile(filepath.FromSlash("testdata/action-info.lores.ivg"))
	if err != nil {
		log.Fatal(err)
	}

	const width = 24
	dst := image.NewAlpha(image.Rect(0, 0, width, width))
	var z iconvg.Rasterizer
	z.SetDstImage(dst, dst.Bounds(), draw.Src)
	if err := iconvg.Decode(&z, ivgData, nil); err != nil {
		log.Fatal(err)
	}

	const asciiArt = ".++8"
	buf := make([]byte, 0, width*(width+1))
	for y := 0; y < width; y++ {
		for x := 0; x < width; x++ {
			a := dst.AlphaAt(x, y).A
			buf = append(buf, asciiArt[a>>6])
		}
		buf = append(buf, '\n')
	}
	os.Stdout.Write(buf)

	// Output:
	// ........................
	// ........................
	// ........++8888++........
	// ......+8888888888+......
	// .....+888888888888+.....
	// ....+88888888888888+....
	// ...+8888888888888888+...
	// ...88888888..88888888...
	// ..+88888888..88888888+..
	// ..+888888888888888888+..
	// ..88888888888888888888..
	// ..888888888..888888888..
	// ..888888888..888888888..
	// ..888888888..888888888..
	// ..+88888888..88888888+..
	// ..+88888888..88888888+..
	// ...88888888..88888888...
	// ...+8888888888888888+...
	// ....+88888888888888+....
	// .....+888888888888+.....
	// ......+8888888888+......
	// ........++8888++........
	// ........................
	// ........................
}
