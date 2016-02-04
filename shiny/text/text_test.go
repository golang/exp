// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"image"
	"io"
	"io/ioutil"
	"math/rand"
	"reflect"
	"testing"

	"golang.org/x/image/math/fixed"
)

// toyFace implements the font.Face interface by measuring every rune's width
// as 1 pixel.
type toyFace struct{}

func (toyFace) Close() error {
	return nil
}

func (toyFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	panic("unimplemented")
}

func (toyFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	panic("unimplemented")
}

func (toyFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	return fixed.I(1), true
}

func (toyFace) Kern(r0, r1 rune) fixed.Int26_6 {
	return 0
}

// iRobot is some text that contains both ASCII and non-ASCII runes.
const iRobot = "\"I, Robot\" in Russian is \"Я, робот\".\nIt's about robots.\n"

func iRobotFrame() *Frame {
	f := new(Frame)
	f.SetFace(toyFace{})
	f.SetMaxWidth(fixed.I(10))
	c := f.NewCaret()
	c.WriteString(iRobot)
	c.Close()
	return f
}

func TestSeek(t *testing.T) {
	f := iRobotFrame()
	c := f.NewCaret()
	defer c.Close()
	rng := rand.New(rand.NewSource(1))
	seen := [1 + len(iRobot)]bool{}
	for i := 0; i < 10*len(iRobot); i++ {
		wantPos := int64(rng.Intn(len(iRobot) + 1))
		gotPos, err := c.Seek(wantPos, SeekSet)
		if err != nil {
			t.Fatalf("i=%d: Seek: %v", i, err)
		}
		if gotPos != wantPos {
			t.Fatalf("i=%d: Seek: got %d, want %d", gotPos, wantPos)
		}
		seen[gotPos] = true

		gotByte, gotErr := c.ReadByte()
		wantByte, wantErr := byte(0), io.EOF
		if gotPos < int64(len(iRobot)) {
			wantByte, wantErr = iRobot[gotPos], nil
		}
		if gotByte != wantByte || gotErr != wantErr {
			t.Fatalf("i=%d: ReadByte: got %d, %v, want %d, %v", i, gotByte, gotErr, wantByte, wantErr)
		}
	}
	for i, s := range seen {
		if !s {
			t.Errorf("randomly generated positions weren't exhaustive: position %d / %d not seen", i, len(iRobot))
		}
	}
}

func TestRead(t *testing.T) {
	f := iRobotFrame()
	c := f.NewCaret()
	defer c.Close()
	asBytes, err := ioutil.ReadAll(c)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	got, want := string(asBytes), iRobot
	if got != want {
		t.Fatalf("\ngot:  %q\nwant: %q", got, want)
	}
}

func TestReadByte(t *testing.T) {
	f := iRobotFrame()
	c := f.NewCaret()
	defer c.Close()
	got, want := []byte(nil), []byte(iRobot)
	for {
		x, err := c.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadByte: %v", err)
		}
		got = append(got, x)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\ngot:  %v\nwant: %v", got, want)
	}
}

func TestReadRune(t *testing.T) {
	f := iRobotFrame()
	c := f.NewCaret()
	defer c.Close()
	got, want := []rune(nil), []rune(iRobot)
	for {
		r, _, err := c.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadRune: %v", err)
		}
		got = append(got, r)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\ngot:  %v\nwant: %v", got, want)
	}
}

// TODO: fuzz-test that all the invariants remain true when modifying a Frame's
// text.
//
// TODO: enumerate all of the invariants, e.g. that the Boxes' i:j ranges do
// not overlap.
