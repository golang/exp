// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/math/f32"
	"golang.org/x/image/vector"
)

const (
	smoothTypeNone = iota
	smoothTypeQuad
	smoothTypeCube
)

// Rasterizer is a Destination that draws an IconVG graphic onto a raster
// image.
//
// The zero value is usable, in that it has no raster image to draw onto, so
// that calling Decode with this Destination is a no-op (other than checking
// the encoded form for errors in the byte code). Call SetDstImage to change
// the raster image, before calling Decode or between calls to Decode.
type Rasterizer struct {
	z vector.Rasterizer

	dst    draw.Image
	r      image.Rectangle
	drawOp draw.Op

	// scale and bias transforms the metadata.ViewBox rectangle to the (0, 0) -
	// (r.Dx(), r.Dy()) rectangle.
	scaleX float32
	biasX  float32
	scaleY float32
	biasY  float32

	metadata Metadata

	lod0 float32
	lod1 float32
	cSel uint8
	nSel uint8

	disabled bool

	firstStartPath  bool
	prevSmoothType  uint8
	prevSmoothPoint f32.Vec2

	fill      image.Image
	flatColor color.RGBA
	flatImage image.Uniform

	cReg [64]color.RGBA
	nReg [64]float32
}

// SetDstImage sets the Rasterizer to draw onto a destination image, given by
// dst and r, with the given compositing operator.
//
// The IconVG graphic (which does not have a fixed size in pixels) will be
// scaled in the X and Y dimensions to fit the rectangle r. The scaling factors
// may differ in the two dimensions.
func (z *Rasterizer) SetDstImage(dst draw.Image, r image.Rectangle, drawOp draw.Op) {
	z.dst = dst
	if r.Empty() {
		r = image.Rectangle{}
	}
	z.r = r
	z.drawOp = drawOp
	z.recalcTransform()
}

// Reset resets the Rasterizer for the given Metadata.
func (z *Rasterizer) Reset(m Metadata) {
	z.metadata = m
	z.lod0 = 0
	z.lod1 = positiveInfinity
	z.cSel = 0
	z.nSel = 0
	z.firstStartPath = true
	z.prevSmoothType = smoothTypeNone
	z.prevSmoothPoint = f32.Vec2{}
	z.cReg = m.Palette
	z.nReg = [64]float32{}
	z.recalcTransform()
}

func (z *Rasterizer) recalcTransform() {
	z.scaleX = float32(z.r.Dx()) / (z.metadata.ViewBox.Max[0] - z.metadata.ViewBox.Min[0])
	z.biasX = -z.metadata.ViewBox.Min[0]
	z.scaleY = float32(z.r.Dy()) / (z.metadata.ViewBox.Max[1] - z.metadata.ViewBox.Min[1])
	z.biasY = -z.metadata.ViewBox.Min[1]
}

func (z *Rasterizer) SetCSel(cSel uint8) { z.cSel = cSel & 0x3f }
func (z *Rasterizer) SetNSel(nSel uint8) { z.nSel = nSel & 0x3f }

func (z *Rasterizer) SetCReg(adj uint8, incr bool, c Color) {
	z.cReg[(z.cSel-adj)&0x3f] = c.Resolve(&z.metadata.Palette, &z.cReg)
	if incr {
		z.cSel++
	}
}

func (z *Rasterizer) SetNReg(adj uint8, incr bool, f float32) {
	z.nReg[(z.nSel-adj)&0x3f] = f
	if incr {
		z.nSel++
	}
}

func (z *Rasterizer) SetLOD(lod0, lod1 float32) {
	z.lod0, z.lod1 = lod0, lod1
}

func (z *Rasterizer) absX(x float32) float32 { return z.scaleX * (x + z.biasX) }
func (z *Rasterizer) absY(y float32) float32 { return z.scaleY * (y + z.biasY) }
func (z *Rasterizer) relX(x float32) float32 { return z.scaleX * x }
func (z *Rasterizer) relY(y float32) float32 { return z.scaleY * y }

func (z *Rasterizer) absVec2(x, y float32) f32.Vec2 {
	return f32.Vec2{z.absX(x), z.absY(y)}
}

func (z *Rasterizer) relVec2(x, y float32) f32.Vec2 {
	pen := z.z.Pen()
	return f32.Vec2{pen[0] + z.relX(x), pen[1] + z.relY(y)}
}

// implicitSmoothPoint returns the implicit control point for smooth-quadratic
// and smooth-cubic Bézier curves.
//
// https://www.w3.org/TR/SVG/paths.html#PathDataCurveCommands says, "The first
// control point is assumed to be the reflection of the second control point on
// the previous command relative to the current point. (If there is no previous
// command or if the previous command was not [a quadratic or cubic command],
// assume the first control point is coincident with the current point.)"
func (z *Rasterizer) implicitSmoothPoint(thisSmoothType uint8) f32.Vec2 {
	pen := z.z.Pen()
	if z.prevSmoothType != thisSmoothType {
		return pen
	}
	return f32.Vec2{
		2*pen[0] - z.prevSmoothPoint[0],
		2*pen[1] - z.prevSmoothPoint[1],
	}
}

func (z *Rasterizer) StartPath(adj uint8, x, y float32) {
	// TODO: gradient fills, not just flat colors.
	z.flatColor = z.cReg[(z.cSel-adj)&0x3f]
	z.flatImage.C = &z.flatColor
	z.fill = &z.flatImage

	width, height := z.r.Dx(), z.r.Dy()
	h := float32(height)
	z.disabled = z.flatColor.A == 0 || !(z.lod0 <= h && h < z.lod1)
	if z.disabled {
		return
	}

	z.z.Reset(width, height)
	if z.firstStartPath {
		z.firstStartPath = false
		z.z.DrawOp = z.drawOp
	}
	z.prevSmoothType = smoothTypeNone
	z.z.MoveTo(z.absVec2(x, y))
}

func (z *Rasterizer) ClosePathEndPath() {
	if z.disabled {
		return
	}
	z.z.ClosePath()
	if z.dst == nil {
		return
	}
	z.z.Draw(z.dst, z.r, z.fill, image.Point{})
}

func (z *Rasterizer) ClosePathAbsMoveTo(x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	z.z.ClosePath()
	z.z.MoveTo(z.absVec2(x, y))
}

func (z *Rasterizer) ClosePathRelMoveTo(x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	z.z.ClosePath()
	z.z.MoveTo(z.relVec2(x, y))
}

func (z *Rasterizer) AbsHLineTo(x float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	pen := z.z.Pen()
	z.z.LineTo(f32.Vec2{z.absX(x), pen[1]})
}

func (z *Rasterizer) RelHLineTo(x float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	pen := z.z.Pen()
	z.z.LineTo(f32.Vec2{pen[0] + z.relX(x), pen[1]})
}

func (z *Rasterizer) AbsVLineTo(y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	pen := z.z.Pen()
	z.z.LineTo(f32.Vec2{pen[0], z.absY(y)})
}

func (z *Rasterizer) RelVLineTo(y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	pen := z.z.Pen()
	z.z.LineTo(f32.Vec2{pen[0], pen[1] + z.relY(y)})
}

func (z *Rasterizer) AbsLineTo(x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	z.z.LineTo(z.absVec2(x, y))
}

func (z *Rasterizer) RelLineTo(x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	z.z.LineTo(z.relVec2(x, y))
}

func (z *Rasterizer) AbsSmoothQuadTo(x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeQuad
	z.prevSmoothPoint = z.implicitSmoothPoint(smoothTypeQuad)
	z.z.QuadTo(z.prevSmoothPoint, z.absVec2(x, y))
}

func (z *Rasterizer) RelSmoothQuadTo(x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeQuad
	z.prevSmoothPoint = z.implicitSmoothPoint(smoothTypeQuad)
	z.z.QuadTo(z.prevSmoothPoint, z.relVec2(x, y))
}

func (z *Rasterizer) AbsQuadTo(x1, y1, x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeQuad
	z.prevSmoothPoint = z.absVec2(x1, y1)
	z.z.QuadTo(z.prevSmoothPoint, z.absVec2(x, y))
}

func (z *Rasterizer) RelQuadTo(x1, y1, x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeQuad
	z.prevSmoothPoint = z.relVec2(x1, y1)
	z.z.QuadTo(z.prevSmoothPoint, z.relVec2(x, y))
}

func (z *Rasterizer) AbsSmoothCubeTo(x2, y2, x, y float32) {
	if z.disabled {
		return
	}
	p1 := z.implicitSmoothPoint(smoothTypeCube)
	z.prevSmoothType = smoothTypeCube
	z.prevSmoothPoint = z.absVec2(x2, y2)
	z.z.CubeTo(p1, z.prevSmoothPoint, z.absVec2(x, y))
}

func (z *Rasterizer) RelSmoothCubeTo(x2, y2, x, y float32) {
	if z.disabled {
		return
	}
	p1 := z.implicitSmoothPoint(smoothTypeCube)
	z.prevSmoothType = smoothTypeCube
	z.prevSmoothPoint = z.relVec2(x2, y2)
	z.z.CubeTo(p1, z.prevSmoothPoint, z.relVec2(x, y))
}

func (z *Rasterizer) AbsCubeTo(x1, y1, x2, y2, x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeCube
	z.prevSmoothPoint = z.absVec2(x2, y2)
	z.z.CubeTo(z.absVec2(x1, y1), z.prevSmoothPoint, z.absVec2(x, y))
}

func (z *Rasterizer) RelCubeTo(x1, y1, x2, y2, x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeCube
	z.prevSmoothPoint = z.relVec2(x2, y2)
	z.z.CubeTo(z.relVec2(x1, y1), z.prevSmoothPoint, z.relVec2(x, y))
}

func (z *Rasterizer) AbsArcTo(rx, ry, xAxisRotation float32, largeArc, sweep bool, x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	// TODO: implement.
}

func (z *Rasterizer) RelArcTo(rx, ry, xAxisRotation float32, largeArc, sweep bool, x, y float32) {
	if z.disabled {
		return
	}
	z.prevSmoothType = smoothTypeNone
	// TODO: implement.
}
