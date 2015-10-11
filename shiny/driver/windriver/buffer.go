// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

import "image"

type bufferImpl struct {
	rgba *image.RGBA
}

func (b *bufferImpl) Release() {
	b.rgba = nil
}

func (b *bufferImpl) Size() image.Point {
	return b.rgba.Rect.Max
}

func (b *bufferImpl) Bounds() image.Rectangle {
	return b.rgba.Rect
}

func (b *bufferImpl) RGBA() *image.RGBA {
	return b.rgba
}
