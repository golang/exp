// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gldriver

import (
	"image"

	"golang.org/x/exp/shiny/screen"
)

type textureImpl struct {
	size image.Point
}

func (t *textureImpl) Release() {
	// TODO
}

func (t *textureImpl) Size() image.Point { return t.size }

func (t *textureImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle, sender screen.Sender) {
}
