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
	"os"
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
	got, err := disassemble(actionInfoIconVG)
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
	got, err := rasterizeASCIIArt(24, actionInfoIconVG)
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
