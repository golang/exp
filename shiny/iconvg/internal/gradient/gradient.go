// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gradient provides linear and radial gradient images.
package gradient

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/math/f64"
)

// TODO: gamma correction / non-linear color interpolation?

// TODO: move this out of an internal directory, either under
// golang.org/x/image or under the standard library's image, so that
// golang.org/x/image/{draw,vector} and possibly image/draw can type switch on
// the gradient.Gradient type and provide fast path code.
//
// Doing so requires coming up with a stable API that we'd be happy to support
// in the long term.

// Shape is the gradient shape.
type Shape uint8

const (
	ShapeLinear Shape = iota
	ShapeRadial
)

// Spread is the gradient spread, or how to spread a gradient past its nominal
// bounds (from offset being 0.0 to offset being 1.0).
type Spread uint8

const (
	// SpreadNone means that offsets outside of the [0, 1] range map to
	// transparent black.
	SpreadNone Spread = iota
	// SpreadPad means that offsets below 0 and above 1 map to the colors that
	// 0 and 1 would map to.
	SpreadPad
	// SpreadReflect means that the offset mapping is reflected start-to-end,
	// end-to-start, start-to-end, etc.
	SpreadReflect
	// SpreadRepeat means that the offset mapping is repeated start-to-end,
	// start-to-end, start-to-end, etc.
	SpreadRepeat
)

// Clamp clamps x to the range [0, 1]. If x is outside that range, it is
// converted to a value in that range according to s's semantics. It returns -1
// if s is SpreadNone and x is outside the range [0, 1].
func (s Spread) Clamp(x float64) float64 {
	if x >= 0 {
		if x <= 1 {
			return x
		}
		switch s {
		case SpreadPad:
			return 1
		case SpreadReflect:
			if int(x)&1 == 0 {
				return x - math.Floor(x)
			}
			return math.Ceil(x) - x
		case SpreadRepeat:
			return x - math.Floor(x)
		}
		return -1
	}
	switch s {
	case SpreadPad:
		return 0
	case SpreadReflect:
		x = -x
		if int(x)&1 == 0 {
			return x - math.Floor(x)
		}
		return math.Ceil(x) - x
	case SpreadRepeat:
		return x - math.Floor(x)
	}
	return -1
}

// Stop is an offset and color.
type Stop struct {
	Offset float64
	RGBA64 color.RGBA64
}

// Range is the range between two stops.
type Range struct {
	Offset0 float64
	Offset1 float64
	Width   float64
	R0      float64
	R1      float64
	G0      float64
	G1      float64
	B0      float64
	B1      float64
	A0      float64
	A1      float64
}

// MakeRange returns the range between two stops.
func MakeRange(s0, s1 Stop) Range {
	return Range{
		Offset0: s0.Offset,
		Offset1: s1.Offset,
		Width:   s1.Offset - s0.Offset,
		R0:      float64(s0.RGBA64.R),
		R1:      float64(s1.RGBA64.R),
		G0:      float64(s0.RGBA64.G),
		G1:      float64(s1.RGBA64.G),
		B0:      float64(s0.RGBA64.B),
		B1:      float64(s1.RGBA64.B),
		A0:      float64(s0.RGBA64.A),
		A1:      float64(s1.RGBA64.A),
	}
}

// AppendRanges appends to a the ranges defined by a's implicit final stop (if
// any exist) and stops.
func AppendRanges(a []Range, stops []Stop) []Range {
	if len(stops) == 0 {
		return nil
	}
	if len(a) != 0 {
		z := a[len(a)-1]
		a = append(a, MakeRange(Stop{
			Offset: z.Offset1,
			RGBA64: color.RGBA64{
				R: uint16(z.R1),
				G: uint16(z.G1),
				B: uint16(z.B1),
				A: uint16(z.A1),
			},
		}, stops[0]))
	}
	for i := 0; i < len(stops)-1; i++ {
		a = append(a, MakeRange(stops[i], stops[i+1]))
	}
	return a
}

// Gradient is a very large image.Image (the same size as an image.Uniform)
// whose colors form a gradient.
type Gradient struct {
	Shape  Shape
	Spread Spread
	Ranges []Range

	// First and Last are the first and last stop's colors.
	First, Last color.RGBA64

	// Pix2Grad transforms coordinates from pixel space (the arguments to the
	// Image.At method) to gradient space. Gradient space is where a linear
	// gradient ranges from x == 0 to x == 1, and a radial gradient has center
	// (0, 0) and radius 1.
	//
	// This is an affine transform, so it can represent elliptical gradients in
	// pixel space, including non-axis-aligned ellipses.
	//
	// For a linear gradient, the bottom row is ignored.
	Pix2Grad f64.Aff3
}

func (g *Gradient) init(spread Spread, stops []Stop) {
	g.Spread = spread
	g.Ranges = AppendRanges(g.Ranges[:0], stops)
	if len(stops) == 0 {
		g.First = color.RGBA64{}
		g.Last = color.RGBA64{}
	} else {
		g.First = stops[0].RGBA64
		g.Last = stops[len(stops)-1].RGBA64
	}
}

// InitLinear initializes g to be a linear gradient from (x1, y1) to (x2, y2),
// in pixel space. Its colors are given by spread and stops.
func (g *Gradient) InitLinear(x1, y1, x2, y2 float64, spread Spread, stops []Stop) {
	g.init(spread, stops)
	g.Shape = ShapeLinear
	dx, dy := x2-x1, y2-y1
	// The top row [a, b, c] of the Pix2Grad matrix satisfies the three
	// simultaneous equations:
	//	a*(x1   ) + b*(y1   ) + c = 0   (eq #0)
	//	a*(x1+dy) + b*(y1-dx) + c = 0   (eq #1)
	//	a*(x1+dx) + b*(y1+dy) + c = 1   (eq #2)
	// Subtracting equation #0 from equations #1 and #2 give:
	//	a*(  +dy) + b*(  -dx)     = 0   (eq #3)
	//	a*(  +dx) + b*(  +dy)     = 1   (eq #4)
	// So that
	//	a*(dy*dy) - b*(dy*dx)     = 0   (eq #5)
	//	a*(dx*dx) + b*(dx*dy)     = dx  (eq #6)
	// And that
	//	a = dx / (dx*dx + dy*dy)        (eq #7)
	// Equations #3 and #7 yield:
	//	b = dy / (dx*dx + dy*dy)        (eq #8)
	d := dx*dx + dy*dy
	a := dx / d
	b := dy / d
	g.Pix2Grad = f64.Aff3{
		a, b, -a*x1 - b*y1,
		0, 0, 0,
	}
}

// InitCircular initializes g to be a circular gradient centered on (cx, cy)
// with radius r, in pixel space. Its colors are given by spread and stops.
func (g *Gradient) InitCircular(cx, cy, r float64, spread Spread, stops []Stop) {
	g.init(spread, stops)
	g.Shape = ShapeRadial
	invR := 1 / r
	g.Pix2Grad = f64.Aff3{
		invR, 0, -cx * invR,
		0, invR, -cy * invR,
	}
}

// TODO: Gradient.InitElliptical?

// ColorModel satisfies the image.Image interface.
func (g *Gradient) ColorModel() color.Model {
	return color.RGBA64Model
}

// Bounds satisfies the image.Image interface.
func (g *Gradient) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{-1e9, -1e9},
		Max: image.Point{+1e9, +1e9},
	}
}

// At satisfies the image.Image interface.
func (g *Gradient) At(x, y int) color.Color {
	if len(g.Ranges) == 0 {
		return color.RGBA64{}
	}

	px := float64(x) + 0.5
	py := float64(y) + 0.5

	offset := 0.0
	if g.Shape == ShapeLinear {
		offset = g.Spread.Clamp(g.Pix2Grad[0]*px + g.Pix2Grad[1]*py + g.Pix2Grad[2])
	} else {
		gx := g.Pix2Grad[0]*px + g.Pix2Grad[1]*py + g.Pix2Grad[2]
		gy := g.Pix2Grad[3]*px + g.Pix2Grad[4]*py + g.Pix2Grad[5]
		offset = g.Spread.Clamp(math.Sqrt(gx*gx + gy*gy))
	}
	if !(offset >= 0) {
		return color.RGBA64{}
	}

	if offset < g.Ranges[0].Offset0 {
		return g.First
	}
	for _, r := range g.Ranges {
		if r.Offset0 <= offset && offset <= r.Offset1 {
			t := (offset - r.Offset0) / r.Width
			s := 1 - t
			return color.RGBA64{
				uint16(s*r.R0 + t*r.R1),
				uint16(s*r.G0 + t*r.G1),
				uint16(s*r.B0 + t*r.B1),
				uint16(s*r.A0 + t*r.A1),
			}
		}
	}
	return g.Last
}
