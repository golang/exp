// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/exp/shiny/screen"
)

type textureImpl struct{}

func (t *textureImpl) Bounds() image.Rectangle {
	return image.Rectangle{}
}

func (t *textureImpl) Fill(r image.Rectangle, c color.Color, op draw.Op) {
	// TODO
}

func (t *textureImpl) Release() {
	// TODO
}

func (t *textureImpl) Size() image.Point {
	// TODO
	return image.Point{}
}

func (t *textureImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle) {
	// TODO
}
