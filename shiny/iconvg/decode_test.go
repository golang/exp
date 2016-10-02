// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// disassemble returns a disassembly of an encoded IconVG graphic. Users of
// this package aren't expected to want to do this, so it lives in a _test.go
// file, but it can be useful for debugging.
func disassemble(src []byte) (string, error) {
	w := new(bytes.Buffer)
	p := func(b []byte, format string, args ...interface{}) {
		const hex = "0123456789abcdef"
		var buf [14]byte
		for i := range buf {
			buf[i] = ' '
		}
		for i, x := range b {
			buf[3*i+0] = hex[x>>4]
			buf[3*i+1] = hex[x&0x0f]
		}
		w.Write(buf[:])
		fmt.Fprintf(w, format, args...)
	}
	m := Metadata{}
	if err := decode(nil, p, &m, false, buffer(src), nil); err != nil {
		return "", err
	}
	return w.String(), nil
}

var (
	_ Destination = (*Encoder)(nil)
	_ Destination = (*Rasterizer)(nil)
)

// encodePNG is useful for manually debugging the tests.
func encodePNG(dstFilename string, src image.Image) error {
	f, err := os.Create(dstFilename)
	if err != nil {
		return err
	}
	encErr := png.Encode(f, src)
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	return closeErr
}

func decodePNG(srcFilename string) (image.Image, error) {
	f, err := os.Open(srcFilename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func checkApproxEqual(m0, m1 image.Image) error {
	diff := func(a, b uint32) uint32 {
		if a < b {
			return b - a
		}
		return a - b
	}

	bounds0 := m0.Bounds()
	bounds1 := m1.Bounds()
	if bounds0 != bounds1 {
		return fmt.Errorf("bounds differ: got %v, want %v", bounds0, bounds1)
	}
	for y := bounds0.Min.Y; y < bounds0.Max.Y; y++ {
		for x := bounds0.Min.X; x < bounds0.Max.X; x++ {
			r0, g0, b0, a0 := m0.At(x, y).RGBA()
			r1, g1, b1, a1 := m1.At(x, y).RGBA()
			const D = 0xffff * 5 / 100 // Diff threshold of 5%.
			if diff(r0, r1) > D || diff(g0, g1) > D || diff(b0, b1) > D || diff(a0, a1) > D {
				return fmt.Errorf("at (%d, %d):\n"+
					"got  RGBA %#04x, %#04x, %#04x, %#04x\n"+
					"want RGBA %#04x, %#04x, %#04x, %#04x",
					x, y, r0, g0, b0, a0, r1, g1, b1, a1)
			}
		}
	}
	return nil
}

func diffLines(t *testing.T, got, want string) {
	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")
	for i := 1; ; i++ {
		if len(gotLines) == 0 {
			t.Errorf("line %d:\ngot  %q\nwant %q", i, "", wantLines[0])
			return
		}
		if len(wantLines) == 0 {
			t.Errorf("line %d:\ngot  %q\nwant %q", i, gotLines[0], "")
			return
		}
		g, w := gotLines[0], wantLines[0]
		gotLines = gotLines[1:]
		wantLines = wantLines[1:]
		if g != w {
			t.Errorf("line %d:\ngot  %q\nwant %q", i, g, w)
			return
		}
	}
}

func rasterizeASCIIArt(width int, encoded []byte) (string, error) {
	dst := image.NewAlpha(image.Rect(0, 0, width, width))
	var z Rasterizer
	z.SetDstImage(dst, dst.Bounds(), draw.Src)
	if err := Decode(&z, encoded, nil); err != nil {
		return "", err
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
	return string(buf), nil
}

func TestDisassembleActionInfo(t *testing.T) {
	ivgData, err := ioutil.ReadFile(filepath.FromSlash("testdata/action-info.ivg"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := disassemble(ivgData)
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"89 49 56 47   Magic identifier",
		"02            Number of metadata chunks: 1",
		"0a            Metadata chunk length: 5",
		"00            Metadata Identifier: 0 (viewBox)",
		"50                -24",
		"50                -24",
		"b0                +24",
		"b0                +24",
		"c0            Start path, filled with CREG[CSEL-0]; M (absolute moveTo)",
		"80                +0",
		"58                -20",
		"a0            C (absolute cubeTo), 1 reps",
		"cf cc 30 c1       -11.049999",
		"58                -20",
		"58                -20",
		"cf cc 30 c1       -11.049999",
		"58                -20",
		"80                +0",
		"91            s (relative smooth cubeTo), 2 reps",
		"37 33 0f 41       +8.950001",
		"a8                +20",
		"a8                +20",
		"a8                +20",
		"              s (relative smooth cubeTo), implicit",
		"a8                +20",
		"37 33 0f c1       -8.950001",
		"a8                +20",
		"58                -20",
		"80            S (absolute smooth cubeTo), 1 reps",
		"cf cc 30 41       +11.049999",
		"58                -20",
		"80                +0",
		"58                -20",
		"e3            z (closePath); m (relative moveTo)",
		"84                +2",
		"bc                +30",
		"e7            h (relative horizontal lineTo)",
		"78                -4",
		"e8            V (absolute vertical lineTo)",
		"7c                -2",
		"e7            h (relative horizontal lineTo)",
		"88                +4",
		"e9            v (relative vertical lineTo)",
		"98                +12",
		"e3            z (closePath); m (relative moveTo)",
		"80                +0",
		"60                -16",
		"e7            h (relative horizontal lineTo)",
		"78                -4",
		"e9            v (relative vertical lineTo)",
		"78                -4",
		"e7            h (relative horizontal lineTo)",
		"88                +4",
		"e9            v (relative vertical lineTo)",
		"88                +4",
		"e1            z (closePath); end path",
	}, "\n") + "\n"

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
		diffLines(t, got, want)
	}
}

func TestDecodeActionInfo(t *testing.T) {
	ivgData, err := ioutil.ReadFile(filepath.FromSlash("testdata/action-info.ivg"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := rasterizeASCIIArt(24, ivgData)
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"........................",
		"........................",
		"........++8888++........",
		"......+8888888888+......",
		".....+888888888888+.....",
		"....+88888888888888+....",
		"...+8888888888888888+...",
		"...88888888..88888888...",
		"..+88888888..88888888+..",
		"..+888888888888888888+..",
		"..88888888888888888888..",
		"..888888888..888888888..",
		"..888888888..888888888..",
		"..888888888..888888888..",
		"..+88888888..88888888+..",
		"..+88888888..88888888+..",
		"...88888888..88888888...",
		"...+8888888888888888+...",
		"....+88888888888888+....",
		".....+888888888888+.....",
		"......+8888888888+......",
		"........++8888++........",
		"........................",
		"........................",
	}, "\n") + "\n"

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
		diffLines(t, got, want)
	}
}

func TestRasterizer(t *testing.T) {
	testCases := []string{
		"testdata/action-info",
		"testdata/video-005.primitive",
	}

	for _, tc := range testCases {
		ivgData, err := ioutil.ReadFile(filepath.FromSlash(tc) + ".ivg")
		if err != nil {
			t.Errorf("%s: ReadFile: %v", tc, err)
			continue
		}
		md, err := DecodeMetadata(ivgData)
		if err != nil {
			t.Errorf("%s: DecodeMetadata: %v", tc, err)
			continue
		}
		width, height := 256, 256
		if dx, dy := md.ViewBox.AspectRatio(); dx < dy {
			width = int(256 * dx / dy)
		} else {
			height = int(256 * dy / dx)
		}

		got := image.NewRGBA(image.Rect(0, 0, width, height))
		var z Rasterizer
		z.SetDstImage(got, got.Bounds(), draw.Src)
		if err := Decode(&z, ivgData, nil); err != nil {
			t.Errorf("%s: Decode: %v", tc, err)
			continue
		}

		want, err := decodePNG(filepath.FromSlash(tc) + ".png")
		if err != nil {
			t.Errorf("%s: decodePNG: %v", tc, err)
			continue
		}
		if err := checkApproxEqual(got, want); err != nil {
			t.Errorf("%s: %v", tc, err)
			continue
		}
	}
}
