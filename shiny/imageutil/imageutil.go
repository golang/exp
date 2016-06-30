// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package imageutil implements some image utility functions.
package imageutil

import (
	"image"
)

// TODO: move Border into the standard library's package image?

// Border returns four rectangles that together contain those points between r
// and r.Inset(inset). Visually:
//
//	00000000
//	00000000
//	11....22
//	11....22
//	11....22
//	33333333
//	33333333
//
// The inset may be negative, in which case the points will be outside r.
//
// Some of the returned rectangles may be empty. None of the returned
// rectangles will overlap.
func Border(r image.Rectangle, inset int) [4]image.Rectangle {
	if inset == 0 {
		return [4]image.Rectangle{}
	}
	if r.Dx() <= 2*inset || r.Dy() <= 2*inset {
		return [4]image.Rectangle{r}
	}

	x := [4]int{
		r.Min.X,
		r.Min.X + inset,
		r.Max.X - inset,
		r.Max.X,
	}
	y := [4]int{
		r.Min.Y,
		r.Min.Y + inset,
		r.Max.Y - inset,
		r.Max.Y,
	}
	if inset < 0 {
		x[0], x[1] = x[1], x[0]
		x[2], x[3] = x[3], x[2]
		y[0], y[1] = y[1], y[0]
		y[2], y[3] = y[3], y[2]
	}

	// The top and bottom sections are responsible for filling the corners.
	// The top and bottom sections go from x[0] to x[3], across the y's.
	// The left and right sections go from y[1] to y[2], across the x's.

	return [4]image.Rectangle{{
		// Top section.
		Min: image.Point{
			X: x[0],
			Y: y[0],
		},
		Max: image.Point{
			X: x[3],
			Y: y[1],
		},
	}, {
		// Left section.
		Min: image.Point{
			X: x[0],
			Y: y[1],
		},
		Max: image.Point{
			X: x[1],
			Y: y[2],
		},
	}, {
		// Right section.
		Min: image.Point{
			X: x[2],
			Y: y[1],
		},
		Max: image.Point{
			X: x[3],
			Y: y[2],
		},
	}, {
		// Bottom section.
		Min: image.Point{
			X: x[0],
			Y: y[2],
		},
		Max: image.Point{
			X: x[3],
			Y: y[3],
		},
	}}
}
