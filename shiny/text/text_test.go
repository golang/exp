// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/image/math/fixed"
)

func readAllText(dst []byte, f *Frame) []byte {
	for p := f.FirstParagraph(); p != nil; p = p.Next(f) {
		for l := p.FirstLine(f); l != nil; l = l.Next(f) {
			for b := l.FirstBox(f); b != nil; b = b.Next(f) {
				dst = append(dst, b.Text(f)...)
			}
		}
	}
	return dst
}

func readAllRunes(dst []rune, rr io.RuneReader) ([]rune, error) {
	for {
		r, size, err := rr.ReadRune()
		if err == io.EOF {
			return dst, nil
		}
		if err != nil {
			return nil, err
		}

		// Check that r and size are consistent.
		if wantSize := utf8.RuneLen(r); size == wantSize {
			// OK.
		} else if r == utf8.RuneError && size == 1 {
			// Also OK; one byte of invalid UTF-8 was replaced by '\ufffd'.
		} else {
			return nil, fmt.Errorf("rune %#x: got size %d, want %d", r, size, wantSize)
		}

		dst = append(dst, r)
	}
}

func runesEqual(xs, ys []rune) bool {
	if len(xs) != len(ys) {
		return false
	}
	for i := range xs {
		if xs[i] != ys[i] {
			return false
		}
	}
	return true
}

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
			t.Fatalf("i=%d: Seek: got %d, want %d", i, gotPos, wantPos)
		}
		seen[gotPos] = true
		if err := checkInvariants(f); err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}

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

func testRead(f *Frame, want string) error {
	c := f.NewCaret()
	defer c.Close()
	asBytes, err := ioutil.ReadAll(c)
	if err != nil {
		return fmt.Errorf("ReadAll: %v", err)
	}
	if got := string(asBytes); got != want {
		return fmt.Errorf("Read\ngot:  %q\nwant: %q", got, want)
	}
	return nil
}

func TestRead(t *testing.T) {
	f := iRobotFrame()
	if err := checkInvariants(f); err != nil {
		t.Fatal(err)
	}
	if err := testRead(f, iRobot); err != nil {
		t.Fatal(err)
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
		if err := checkInvariants(f); err != nil {
			t.Fatal(err)
		}
		got = append(got, x)
	}
	if err := checkInvariants(f); err != nil {
		t.Fatal(err)
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
		if err := checkInvariants(f); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if err := checkInvariants(f); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\ngot:  %v\nwant: %v", got, want)
	}
}

func TestWrite(t *testing.T) {
	f := new(Frame)
	c := f.NewCaret()
	c.Write([]byte{0xff, 0xfe, 0xfd})
	c.WriteByte(0x80)
	c.WriteRune('\U0001f4a9')
	c.WriteString("abc\x00")
	c.Close()
	if err := checkInvariants(f); err != nil {
		t.Fatal(err)
	}
	got := f.text[:f.len]
	want := []byte{0xff, 0xfe, 0xfd, 0x80, 0xf0, 0x9f, 0x92, 0xa9, 0x61, 0x62, 0x63, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("\ngot  % x\nwant % x", got, want)
	}
}

func TestSetMaxWidth(t *testing.T) {
	f := iRobotFrame()
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		f.SetMaxWidth(fixed.I(rng.Intn(20)))
		if err := checkInvariants(f); err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}
		if err := testRead(f, iRobot); err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}
	}
}

func TestReadRuneAcrossBoxes(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 6; i++ {
		// text is a mixture of valid and invalid UTF-8.
		text := "ab_\u0100\u0101_\ufffd中文_\xff\xffñ\xff_z_\U0001f4a9_\U0001f600" +
			strings.Repeat("\xff", i)

		wantBytes := []byte(text)
		wantRunes := []rune(text)
		gotBytesBuf := make([]byte, 0, len(text))
		gotRunesBuf := make([]rune, 0, len(text))

		for j := 0; j < 100; j++ {
			f := new(Frame)

			{
				c := f.NewCaret()
				c.WriteString(text)
				// Split the sole Line into multiple Boxes at random byte offsets,
				// possibly cutting across multi-byte UTF-8 encoded runes.
				for {
					if rng.Intn(8) == 0 {
						break
					}
					c.Seek(int64(rng.Intn(len(text)+1)), SeekSet)
					c.splitBox()
				}
				c.Close()

				if err := checkInvariants(f); err != nil {
					t.Fatalf("i=%d, j=%d: %v", i, j, err)
				}
				if gotBytes := readAllText(gotBytesBuf[:0], f); !bytes.Equal(gotBytes, wantBytes) {
					t.Fatalf("i=%d, j=%d: bytes\ngot  % x\nwant % x", i, j, gotBytes, wantBytes)
				}
			}

			// Test lineReader.ReadRune.
			{
				p := f.firstP
				l := f.paragraphs[p].firstL
				b := f.lines[l].firstB
				lr := f.lineReader(b, f.boxes[b].i)
				gotRunes, err := readAllRunes(gotRunesBuf[:0], lr)
				if err != nil {
					t.Fatalf("i=%d, j=%d: lineReader readAllRunes: %v", i, j, err)
				}
				if !runesEqual(gotRunes, wantRunes) {
					t.Fatalf("i=%d, j=%d: lineReader readAllRunes:\ngot  %#x\nwant %#x",
						i, j, gotRunes, wantRunes)
				}
			}

			// Test Caret.ReadRune.
			{
				c := f.NewCaret()
				gotRunes, err := readAllRunes(gotRunesBuf[:0], c)
				c.Close()
				if err != nil {
					t.Fatalf("i=%d, j=%d: Caret readAllRunes: %v", i, j, err)
				}
				if !runesEqual(gotRunes, wantRunes) {
					t.Fatalf("i=%d, j=%d: Caret readAllRunes:\ngot  %#x\nwant %#x",
						i, j, gotRunes, wantRunes)
				}
			}
		}
	}
}

// TODO: fuzz-test that all the invariants remain true when modifying a Frame's
// text.

type ijRange struct {
	i, j int32
}

type byI []ijRange

func (b byI) Len() int           { return len(b) }
func (b byI) Less(x, y int) bool { return b[x].i < b[y].i }
func (b byI) Swap(x, y int)      { b[x], b[y] = b[y], b[x] }

// TODO: ensure that checkInvariants accepts a zero-valued Frame.
func checkInvariants(f *Frame) error {
	const infinity = 1e6

	// Iterate through the Paragraphs, Lines and Boxes. Check that every first
	// child has no previous sibling, and no child is the first child of
	// multiple parents.
	nUsedParagraphs, nUsedLines, nUsedBoxes := 0, 0, 0
	{
		firstLines := map[int32]bool{}
		firstBoxes := map[int32]bool{}
		p := f.firstP
		if p == 0 {
			return fmt.Errorf("firstP is zero")
		}
		if x := f.paragraphs[p].prev; x != 0 {
			return fmt.Errorf("first Paragraph %d's prev: got %d, want 0", p, x)
		}

		for ; p != 0; p = f.paragraphs[p].next {
			l := f.paragraphs[p].firstL
			if l == 0 {
				return fmt.Errorf("paragraphs[%d].firstL is zero", p)
			}
			if x := f.lines[l].prev; x != 0 {
				return fmt.Errorf("first Line %d's prev: got %d, want 0", l, x)
			}
			if firstLines[l] {
				return fmt.Errorf("duplicate first Line %d", l)
			}
			firstLines[l] = true

			for ; l != 0; l = f.lines[l].next {
				b := f.lines[l].firstB
				if b == 0 {
					return fmt.Errorf("lines[%d].firstB is zero", l)
				}
				if x := f.boxes[b].prev; x != 0 {
					return fmt.Errorf("first Box %d's prev: got %d, want 0", b, x)
				}
				if firstBoxes[b] {
					return fmt.Errorf("duplicate first Box %d", b)
				}
				firstBoxes[b] = true

				for ; b != 0; b = f.boxes[b].next {
					nUsedBoxes++
					if nUsedBoxes >= infinity {
						return fmt.Errorf("too many used Boxes (infinite loop?)")
					}
				}
				nUsedLines++
			}
			nUsedParagraphs++
		}
	}

	// Check the paragraphs.
	for p, pp := range f.paragraphs {
		if p == 0 {
			if pp != (Paragraph{}) {
				return fmt.Errorf("paragraphs[0] is a non-zero Paragraph: %v", pp)
			}
			continue
		}
		if pp.next < 0 || len(f.paragraphs) <= int(pp.next) {
			return fmt.Errorf("invalid paragraphs[%d].next: got %d, want in [0, %d)", p, pp.next, len(f.paragraphs))
		}
		if len(f.paragraphs) <= int(pp.prev) {
			return fmt.Errorf("invalid paragraphs[%d].prev: got %d, want in [0, %d)", p, pp.prev, len(f.paragraphs))
		}
		if pp.prev < 0 {
			// The Paragraph is in the free-list, which is checked separately below.
			continue
		}
		if pp.next != 0 && f.paragraphs[pp.next].prev != int32(p) {
			return fmt.Errorf("invalid links: paragraphs[%d].next=%d, paragraphs[%d].prev=%d",
				p, pp.next, pp.next, f.paragraphs[pp.next].prev)
		}
		if pp.prev != 0 && f.paragraphs[pp.prev].next != int32(p) {
			return fmt.Errorf("invalid links: paragraphs[%d].prev=%d, paragraphs[%d].next=%d",
				p, pp.prev, pp.prev, f.paragraphs[pp.prev].next)
		}
	}

	// Check the paragraphs' free-list.
	nFreeParagraphs := 0
	for p := f.freeP; p != 0; nFreeParagraphs++ {
		if nFreeParagraphs >= infinity {
			return fmt.Errorf("Paragraph free-list is too long (infinite loop?)")
		}
		if p < 0 || len(f.paragraphs) <= int(p) {
			return fmt.Errorf("invalid Paragraph free-list index: got %d, want in [0, %d)", p, len(f.paragraphs))
		}
		pp := &f.paragraphs[p]
		if pp.prev >= 0 {
			return fmt.Errorf("paragraphs[%d] is an invalid free-list element: %#v", p, *pp)
		}
		p = pp.next
	}

	// Check the lines.
	for l, ll := range f.lines {
		if l == 0 {
			if ll != (Line{}) {
				return fmt.Errorf("lines[0] is a non-zero Line: %v", ll)
			}
			continue
		}
		if ll.next < 0 || len(f.lines) <= int(ll.next) {
			return fmt.Errorf("invalid lines[%d].next: got %d, want in [0, %d)", l, ll.next, len(f.lines))
		}
		if len(f.lines) <= int(ll.prev) {
			return fmt.Errorf("invalid lines[%d].prev: got %d, want in [0, %d)", l, ll.prev, len(f.lines))
		}
		if ll.prev < 0 {
			// The Line is in the free-list, which is checked separately below.
			continue
		}
		if ll.next != 0 && f.lines[ll.next].prev != int32(l) {
			return fmt.Errorf("invalid links: lines[%d].next=%d, lines[%d].prev=%d",
				l, ll.next, ll.next, f.lines[ll.next].prev)
		}
		if ll.prev != 0 && f.lines[ll.prev].next != int32(l) {
			return fmt.Errorf("invalid links: lines[%d].prev=%d, lines[%d].next=%d",
				l, ll.prev, ll.prev, f.lines[ll.prev].next)
		}
	}

	// Check the lines' free-list.
	nFreeLines := 0
	for l := f.freeL; l != 0; nFreeLines++ {
		if nFreeLines >= infinity {
			return fmt.Errorf("Line free-list is too long (infinite loop?)")
		}
		if l < 0 || len(f.lines) <= int(l) {
			return fmt.Errorf("invalid Line free-list index: got %d, want in [0, %d)", l, len(f.lines))
		}
		ll := &f.lines[l]
		if ll.prev >= 0 {
			return fmt.Errorf("lines[%d] is an invalid free-list element: %#v", l, *ll)
		}
		l = ll.next
	}

	// Check the boxes.
	for b, bb := range f.boxes {
		if b == 0 {
			if bb != (Box{}) {
				return fmt.Errorf("boxes[0] is a non-zero Box: %v", bb)
			}
			continue
		}
		if bb.next < 0 || len(f.boxes) <= int(bb.next) {
			return fmt.Errorf("invalid boxes[%d].next: got %d, want in [0, %d)", b, bb.next, len(f.boxes))
		}
		if len(f.boxes) <= int(bb.prev) {
			return fmt.Errorf("invalid boxes[%d].prev: got %d, want in [0, %d)", b, bb.prev, len(f.boxes))
		}
		if bb.prev < 0 {
			// The Box is in the free-list, which is checked separately below.
			continue
		}
		if bb.next != 0 && f.boxes[bb.next].prev != int32(b) {
			return fmt.Errorf("invalid links: boxes[%d].next=%d, boxes[%d].prev=%d",
				b, bb.next, bb.next, f.boxes[bb.next].prev)
		}
		if bb.prev != 0 && f.boxes[bb.prev].next != int32(b) {
			return fmt.Errorf("invalid links: boxes[%d].prev=%d, boxes[%d].next=%d",
				b, bb.prev, bb.prev, f.boxes[bb.prev].next)
		}
		if 0 > bb.i || bb.i > bb.j || bb.j > int32(len(f.text)) {
			return fmt.Errorf("invalid boxes[%d] i/j range: i=%d, j=%d, len=%d", b, bb.i, bb.j, len(f.text))
		}
	}

	// Check the boxes' free-list.
	nFreeBoxes := 0
	for b := f.freeB; b != 0; nFreeBoxes++ {
		if nFreeBoxes >= infinity {
			return fmt.Errorf("Box free-list is too long (infinite loop?)")
		}
		if b < 0 || len(f.boxes) <= int(b) {
			return fmt.Errorf("invalid Box free-list index: got %d, want in [0, %d)", b, len(f.boxes))
		}
		bb := &f.boxes[b]
		if bb.i != 0 || bb.j != 0 || bb.prev >= 0 {
			return fmt.Errorf("boxes[%d] is an invalid free-list element: %#v", b, *bb)
		}
		b = bb.next
	}

	// Check that the boxes' i:j ranges do not overlap, and their total length
	// equals f.len.
	nText, ijRanges := 0, []ijRange{}
	for _, bb := range f.boxes {
		if bb.i < bb.j {
			nText += int(bb.j - bb.i)
			ijRanges = append(ijRanges, ijRange{i: bb.i, j: bb.j})
		}
	}
	sort.Sort(byI(ijRanges))
	for x := range ijRanges {
		if x == 0 {
			continue
		}
		if ijRanges[x-1].j > ijRanges[x].i {
			return fmt.Errorf("overlapping Box i:j ranges: %v and %v", ijRanges[x-1], ijRanges[x])
		}
	}
	if nText != f.len {
		return fmt.Errorf("text length: got %d, want %d", nText, f.len)
	}

	// Check that every Paragraph, Line and Box, other than the 0th of each, is
	// either used or free.
	if len(f.paragraphs) != 1+nUsedParagraphs+nFreeParagraphs {
		return fmt.Errorf("#paragraphs (%d) != 1 + #used (%d) + #free (%d)",
			len(f.paragraphs), nUsedParagraphs, nFreeParagraphs)
	}
	if len(f.lines) != 1+nUsedLines+nFreeLines {
		return fmt.Errorf("#lines (%d) != 1 + #used (%d) + #free (%d)", len(f.lines), nUsedLines, nFreeLines)
	}
	if len(f.boxes) != 1+nUsedBoxes+nFreeBoxes {
		return fmt.Errorf("#boxes (%d) != 1 + #used (%d) + #free (%d)", len(f.boxes), nUsedBoxes, nFreeBoxes)
	}

	return nil
}

// dump is used for debugging.
func dump(w io.Writer, f *Frame) {
	for p := f.FirstParagraph(); p != nil; p = p.Next(f) {
		for l := p.FirstLine(f); l != nil; l = l.Next(f) {
			for b := l.FirstBox(f); b != nil; b = b.Next(f) {
				fmt.Fprintf(w, "[%s]", b.TrimmedText(f))
			}
			fmt.Fprintln(w)
		}
	}
}
