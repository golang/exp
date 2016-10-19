// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
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
func disassemble(src []byte) ([]byte, error) {
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
		return nil, err
	}
	return w.Bytes(), nil
}

var (
	_ Destination = (*Encoder)(nil)
	_ Destination = (*Rasterizer)(nil)
)

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

			// TODO: be more principled in picking this magic threshold, other
			// than what the difference is, in practice, in x/image/vector's
			// fixed and floating point rasterizer?
			const D = 0xffff * 12 / 100 // Diff threshold of 12%.

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

var testdataTestCases = []struct {
	filename string
	variants string
}{
	{"testdata/action-info.lores", ""},
	{"testdata/action-info.hires", ""},
	{"testdata/arcs", ""},
	{"testdata/blank", ""},
	{"testdata/cowbell", ""},
	{"testdata/elliptical", ""},
	{"testdata/favicon", ";pink"},
	{"testdata/gradient", ""},
	{"testdata/lod-polygon", ";64"},
	{"testdata/video-005.primitive", ""},
}

func TestDisassembly(t *testing.T) {
	for _, tc := range testdataTestCases {
		ivgData, err := ioutil.ReadFile(filepath.FromSlash(tc.filename) + ".ivg")
		if err != nil {
			t.Errorf("%s: ReadFile: %v", tc.filename, err)
			continue
		}
		got, err := disassemble(ivgData)
		if err != nil {
			t.Errorf("%s: disassemble: %v", tc.filename, err)
			continue
		}
		wantFilename := filepath.FromSlash(tc.filename) + ".ivg.disassembly"
		if overwriteTestdataFiles {
			if err := ioutil.WriteFile(filepath.FromSlash(wantFilename), got, 0666); err != nil {
				t.Errorf("%s: WriteFile: %v", tc.filename, err)
			}
			continue
		}
		want, err := ioutil.ReadFile(wantFilename)
		if err != nil {
			t.Errorf("%s: ReadFile: %v", tc.filename, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: got:\n%s\nwant:\n%s", tc.filename, got, want)
			diffLines(t, string(got), string(want))
		}
	}
}

func TestRasterizer(t *testing.T) {
	for _, tc := range testdataTestCases {
		ivgData, err := ioutil.ReadFile(filepath.FromSlash(tc.filename) + ".ivg")
		if err != nil {
			t.Errorf("%s: ReadFile: %v", tc.filename, err)
			continue
		}
		md, err := DecodeMetadata(ivgData)
		if err != nil {
			t.Errorf("%s: DecodeMetadata: %v", tc.filename, err)
			continue
		}

		for _, variant := range strings.Split(tc.variants, ";") {
			length := 256
			if variant == "64" {
				length = 64
			}
			width, height := length, length
			if dx, dy := md.ViewBox.AspectRatio(); dx < dy {
				width = int(float32(length) * dx / dy)
			} else {
				height = int(float32(length) * dy / dx)
			}

			opts := &DecodeOptions{}
			if variant == "pink" {
				pal := DefaultPalette
				pal[0] = color.RGBA{0xfe, 0x76, 0xea, 0xff}
				opts.Palette = &pal
			}

			got := image.NewRGBA(image.Rect(0, 0, width, height))
			var z Rasterizer
			z.SetDstImage(got, got.Bounds(), draw.Src)
			if err := Decode(&z, ivgData, opts); err != nil {
				t.Errorf("%s %q variant: Decode: %v", tc.filename, variant, err)
				continue
			}

			wantFilename := filepath.FromSlash(tc.filename)
			if variant != "" {
				wantFilename += "." + variant
			}
			wantFilename += ".png"
			if overwriteTestdataFiles {
				if err := encodePNG(filepath.FromSlash(wantFilename), got); err != nil {
					t.Errorf("%s %q variant: encodePNG: %v", tc.filename, variant, err)
				}
				continue
			}
			want, err := decodePNG(wantFilename)
			if err != nil {
				t.Errorf("%s %q variant: decodePNG: %v", tc.filename, variant, err)
				continue
			}
			if err := checkApproxEqual(got, want); err != nil {
				t.Errorf("%s %q variant: %v", tc.filename, variant, err)
				continue
			}
		}
	}
}

func TestBlendColor(t *testing.T) {
	// This example comes from doc.go. Look for "orange" in the "Colors"
	// section.
	pal := Palette{
		2: color.RGBA{0xff, 0xcc, 0x80, 0xff}, // "Material Design Orange 200".
	}
	cReg := [64]color.RGBA{}
	got := BlendColor(0x40, 0x7f, 0x82).Resolve(&pal, &cReg)
	want := color.RGBA{0x40, 0x33, 0x20, 0x40} // 25% opaque "Orange 200", alpha-premultiplied.
	if got != want {
		t.Errorf("\ngot  %x\nwant %x", got, want)
	}
}
