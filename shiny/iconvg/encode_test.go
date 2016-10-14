// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg

import (
	"bytes"
	"image/color"
	"io/ioutil"
	"math"
	"path/filepath"
	"testing"

	"golang.org/x/image/math/f32"
)

// overwriteTestdataFiles is temporarily set to true when adding new
// testdataTestCases.
const overwriteTestdataFiles = false

// TestOverwriteTestdataFilesIsFalse tests that any change to
// overwriteTestdataFiles is only temporary. Programmers are assumed to run "go
// test" before sending out for code review or committing code.
func TestOverwriteTestdataFilesIsFalse(t *testing.T) {
	if overwriteTestdataFiles {
		t.Errorf("overwriteTestdataFiles is true; do not commit code changes")
	}
}

func testEncode(t *testing.T, e *Encoder, wantFilename string) {
	got, err := e.Bytes()
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	if overwriteTestdataFiles {
		if err := ioutil.WriteFile(filepath.FromSlash(wantFilename), got, 0666); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return
	}
	want, err := ioutil.ReadFile(filepath.FromSlash(wantFilename))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("\ngot  %d bytes:\n% x\nwant %d bytes:\n% x", len(got), got, len(want), want)
	}
}

func TestEncodeBlank(t *testing.T) {
	var e Encoder
	testEncode(t, &e, "testdata/blank.ivg")
}

func TestEncodeActionInfo(t *testing.T) {
	for _, res := range []string{"lores", "hires"} {
		var e Encoder
		e.Reset(Metadata{
			ViewBox: Rectangle{
				Min: f32.Vec2{-24, -24},
				Max: f32.Vec2{+24, +24},
			},
			Palette: DefaultPalette,
		})
		e.HighResolutionCoordinates = res == "hires"

		e.StartPath(0, 0, -20)
		e.AbsCubeTo(-11.05, -20, -20, -11.05, -20, 0)
		e.RelSmoothCubeTo(8.95, 20, 20, 20)
		e.RelSmoothCubeTo(20, -8.95, 20, -20)
		e.AbsSmoothCubeTo(11.05, -20, 0, -20)
		e.ClosePathRelMoveTo(2, 30)
		e.RelHLineTo(-4)
		e.AbsVLineTo(-2)
		e.RelHLineTo(4)
		e.RelVLineTo(12)
		e.ClosePathRelMoveTo(0, -16)
		e.RelHLineTo(-4)
		e.RelVLineTo(-4)
		e.RelHLineTo(4)
		e.RelVLineTo(4)
		e.ClosePathEndPath()

		testEncode(t, &e, "testdata/action-info."+res+".ivg")
	}
}

func TestEncodeArcs(t *testing.T) {
	var e Encoder

	e.SetCReg(1, false, RGBAColor(color.RGBA{0xff, 0x00, 0x00, 0xff}))
	e.SetCReg(2, false, RGBAColor(color.RGBA{0xff, 0xff, 0x00, 0xff}))
	e.SetCReg(3, false, RGBAColor(color.RGBA{0x00, 0x00, 0x00, 0xff}))
	e.SetCReg(4, false, RGBAColor(color.RGBA{0x00, 0x00, 0x80, 0xff}))

	e.StartPath(1, -10, 0)
	e.RelHLineTo(-15)
	e.RelArcTo(15, 15, 0, true, false, 15, -15)
	e.ClosePathEndPath()

	e.StartPath(2, -14, -4)
	e.RelVLineTo(-15)
	e.RelArcTo(15, 15, 0, false, false, -15, 15)
	e.ClosePathEndPath()

	const thirtyDegrees = 30.0 / 360
	e.StartPath(3, -15, 30)
	e.RelLineTo(5.0, -2.5)
	e.RelArcTo(2.5, 2.5, -thirtyDegrees, false, true, 5.0, -2.5)
	e.RelLineTo(5.0, -2.5)
	e.RelArcTo(2.5, 5.0, -thirtyDegrees, false, true, 5.0, -2.5)
	e.RelLineTo(5.0, -2.5)
	e.RelArcTo(2.5, 7.5, -thirtyDegrees, false, true, 5.0, -2.5)
	e.RelLineTo(5.0, -2.5)
	e.RelArcTo(2.5, 10.0, -thirtyDegrees, false, true, 5.0, -2.5)
	e.RelLineTo(5.0, -2.5)
	e.AbsVLineTo(30)
	e.ClosePathEndPath()

	for largeArc := 0; largeArc <= 1; largeArc++ {
		for sweep := 0; sweep <= 1; sweep++ {
			e.StartPath(4, 10+8*float32(sweep), -28+8*float32(largeArc))
			e.RelArcTo(6, 3, 0, largeArc != 0, sweep != 0, 6, 3)
			e.ClosePathEndPath()
		}
	}

	testEncode(t, &e, "testdata/arcs.ivg")
}

var video005PrimitiveSVGData = []struct {
	r, g, b uint32
	x0, y0  int
	x1, y1  int
	x2, y2  int
}{
	{0x17, 0x06, 0x05, 162, 207, 271, 186, 195, -16},
	{0xe9, 0xf5, 0xf8, -16, 179, 140, -11, 16, -8},
	{0x00, 0x04, 0x27, 97, 96, 221, 21, 214, 111},
	{0x89, 0xd9, 0xff, 262, -6, 271, 104, 164, -16},
	{0x94, 0xbd, 0xc5, 204, 104, 164, 207, 59, 104},
	{0xd4, 0x81, 0x3d, -16, 36, 123, 195, -16, 194},
	{0x00, 0x00, 0x00, 164, 19, 95, 77, 138, 13},
	{0x39, 0x11, 0x19, 50, 143, 115, 185, -4, 165},
	{0x00, 0x3d, 0x81, 86, 109, 53, 76, 90, 24},
	{0xfc, 0xc6, 0x9c, 31, 161, 80, 105, -16, 28},
	{0x9e, 0xdd, 0xff, 201, -7, 31, -16, 2, 60},
	{0x01, 0x20, 0x39, 132, 85, 240, -5, 173, 130},
	{0xfd, 0xbc, 0x8f, 193, 127, 231, 94, 250, 124},
	{0x43, 0x06, 0x00, 251, 207, 237, 83, 271, 97},
	{0x80, 0xbf, 0xee, 117, 134, 88, 177, 90, 28},
	{0x00, 0x00, 0x00, 127, 38, 172, 68, 223, 55},
	{0x19, 0x0e, 0x16, 201, 204, 161, 101, 271, 192},
	{0xf6, 0xaa, 0x71, 201, 164, 226, 141, 261, 152},
	{0xe0, 0x36, 0x00, -16, -2, 29, -16, -6, 58},
	{0xff, 0xe4, 0xba, 146, 45, 118, 75, 148, 76},
	{0x00, 0x00, 0x12, 118, 44, 107, 109, 100, 51},
	{0xbd, 0xd5, 0xe4, 271, 41, 253, -16, 211, 89},
	{0x52, 0x00, 0x00, 87, 127, 83, 150, 55, 111},
	{0x00, 0xb3, 0xa1, 124, 185, 135, 207, 194, 176},
	{0x22, 0x00, 0x00, 59, 151, 33, 124, 52, 169},
	{0xbe, 0xcb, 0xcb, 149, 42, 183, -16, 178, 47},
	{0xff, 0xd4, 0xb1, 211, 119, 184, 100, 182, 124},
	{0xff, 0xe1, 0x39, 73, 207, 140, 180, -13, 187},
	{0xa7, 0xb0, 0xad, 122, 181, 200, 182, 93, 82},
	{0x00, 0x00, 0x00, 271, 168, 170, 185, 221, 207},
}

func TestEncodeVideo005Primitive(t *testing.T) {
	// The division by 4 is because the SVG width is 256 units and the IconVG
	// width is 64 (from -32 to +32).
	//
	// The subtraction by 0.5 is because the SVG file contains the line:
	// <g transform="translate(0.5 0.5)">
	scaleX := func(i int) float32 { return float32(i)/4 - (32 - 0.5/4) }
	scaleY := func(i int) float32 { return float32(i)/4 - (24 - 0.5/4) }

	var e Encoder
	e.Reset(Metadata{
		ViewBox: Rectangle{
			Min: f32.Vec2{-32, -24},
			Max: f32.Vec2{+32, +24},
		},
		Palette: DefaultPalette,
	})

	e.SetCReg(0, false, RGBAColor(color.RGBA{0x7c, 0x7e, 0x7c, 0xff}))
	e.StartPath(0, -32, -24)
	e.AbsHLineTo(+32)
	e.AbsVLineTo(+24)
	e.AbsHLineTo(-32)
	e.ClosePathEndPath()

	for _, v := range video005PrimitiveSVGData {
		e.SetCReg(0, false, RGBAColor(color.RGBA{
			uint8(v.r * 128 / 255),
			uint8(v.g * 128 / 255),
			uint8(v.b * 128 / 255),
			128,
		}))
		e.StartPath(0, scaleX(v.x0), scaleY(v.y0))
		e.AbsLineTo(scaleX(v.x1), scaleY(v.y1))
		e.AbsLineTo(scaleX(v.x2), scaleY(v.y2))
		e.ClosePathEndPath()
	}

	testEncode(t, &e, "testdata/video-005.primitive.ivg")
}

func TestEncodeLODPolygon(t *testing.T) {
	var e Encoder

	poly := func(n int) {
		const r = 28
		angle := 2 * math.Pi / float64(n)
		e.StartPath(0, r, 0)
		for i := 1; i < n; i++ {
			e.AbsLineTo(
				float32(r*math.Cos(angle*float64(i))),
				float32(r*math.Sin(angle*float64(i))),
			)
		}
		e.ClosePathEndPath()
	}

	e.StartPath(0, -28, -20)
	e.AbsVLineTo(-28)
	e.AbsHLineTo(-20)
	e.ClosePathEndPath()

	e.SetLOD(0, 80)
	poly(3)

	e.SetLOD(80, positiveInfinity)
	poly(5)

	e.SetLOD(0, positiveInfinity)
	e.StartPath(0, +28, +20)
	e.AbsVLineTo(+28)
	e.AbsHLineTo(+20)
	e.ClosePathEndPath()

	testEncode(t, &e, "testdata/lod-polygon.ivg")
}
