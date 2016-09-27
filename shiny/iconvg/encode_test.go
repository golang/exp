// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg

import (
	"bytes"
	"testing"

	"golang.org/x/image/math/f32"
)

func TestEncoderZeroValue(t *testing.T) {
	var e Encoder
	got, err := e.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	want := []byte{
		0x89, 0x49, 0x56, 0x47, // Magic identifier.
		0x00, // Zero metadata chunks.
	}
	if !bytes.Equal(got, want) {
		t.Errorf("\ngot  %d bytes:\n% x\nwant %d bytes:\n% x", len(got), got, len(want), want)
	}
}

// actionInfoIconVG is the IconVG encoding of the "action/info" icon from the
// Material Design icon set.
//
// See doc.go for an annotated version.
var actionInfoIconVG = []byte{
	0x89, 0x49, 0x56, 0x47, 0x02, 0x0a, 0x00, 0x50, 0x50, 0xb0, 0xb0, 0xc0, 0x80, 0x58, 0xa0, 0xcf,
	0xcc, 0x30, 0xc1, 0x58, 0x58, 0xcf, 0xcc, 0x30, 0xc1, 0x58, 0x80, 0x91, 0x37, 0x33, 0x0f, 0x41,
	0xa8, 0xa8, 0xa8, 0xa8, 0x37, 0x33, 0x0f, 0xc1, 0xa8, 0x58, 0x80, 0xcf, 0xcc, 0x30, 0x41, 0x58,
	0x80, 0x58, 0xe3, 0x84, 0xbc, 0xe7, 0x78, 0xe8, 0x7c, 0xe7, 0x88, 0xe9, 0x98, 0xe3, 0x80, 0x60,
	0xe7, 0x78, 0xe9, 0x78, 0xe7, 0x88, 0xe9, 0x88, 0xe1,
}

func TestEncodeActionInfo(t *testing.T) {
	var e Encoder
	e.Reset(Metadata{
		ViewBox: Rectangle{
			Min: f32.Vec2{-24, -24},
			Max: f32.Vec2{+24, +24},
		},
		Palette: DefaultPalette,
	})

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

	got, err := e.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	want := actionInfoIconVG
	if !bytes.Equal(got, want) {
		t.Errorf("\ngot  %d bytes:\n% x\nwant %d bytes:\n% x", len(got), got, len(want), want)
	}
}
