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

func rngIntPair(rng *rand.Rand, n int) (x, y int) {
	x = rng.Intn(n)
	y = rng.Intn(n)
	if x > y {
		x, y = y, x
	}
	return x, y
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

func iRobotFrame(maxWidth int) *Frame {
	f := new(Frame)
	f.SetFace(toyFace{})
	f.SetMaxWidth(fixed.I(maxWidth))
	c := f.NewCaret()
	c.WriteString(iRobot)
	c.Close()
	return f
}

func TestZeroFrame(t *testing.T) {
	f := new(Frame)
	if err := checkInvariants(f); err != nil {
		t.Fatal(err)
	}
}

func TestSeek(t *testing.T) {
	f := iRobotFrame(10)
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
	f := iRobotFrame(10)
	if err := checkInvariants(f); err != nil {
		t.Fatal(err)
	}
	if err := testRead(f, iRobot); err != nil {
		t.Fatal(err)
	}
}

func TestReadByte(t *testing.T) {
	f := iRobotFrame(10)
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
	f := iRobotFrame(10)
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

func TestManyWrites(t *testing.T) {
	f := new(Frame)
	f.SetFace(toyFace{})
	f.SetMaxWidth(fixed.I(10))
	c := f.NewCaret()
	defer c.Close()

	const n, abc = 100, "abcdefghijkl\n   "
	rng := rand.New(rand.NewSource(1))
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = abc[rng.Intn(len(abc))]
	}

	for i := 0; i < 100; i++ {
		x, y := rngIntPair(rng, len(buf)+1)
		c.Write(buf[x:y])
		if err := checkInvariants(f); err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}
	}
}

func TestRandomAccessWrite(t *testing.T) {
	f := new(Frame)
	c := f.NewCaret()
	defer c.Close()

	rng := rand.New(rand.NewSource(1))
	gotBuf := make([]byte, 0, 10000)
	want := make([]byte, 0, 10000)
	buf := make([]byte, 10)
	x := byte(0)
	for i := 0; i < 100; i++ {
		n := rng.Intn(len(buf) + 1)
		for j := range buf[:n] {
			buf[j] = x
			x++
		}
		offset := rng.Intn(len(want) + 1)

		// Insert buf[:n] into the Frame at the given offset.
		c.Seek(int64(offset), SeekSet)
		c.Write(buf[:n])
		if err := checkInvariants(f); err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}

		// Insert buf[:n] into want at the given offset.
		want = want[:len(want)+n]
		copy(want[offset+n:], want[offset:])
		copy(want[offset:offset+n], buf[:n])

		if got := readAllText(gotBuf[:0], f); !bytes.Equal(got, want) {
			t.Errorf("i=%d:\ngot  % x\nwant % x", i, got, want)
		}
	}
}

func TestWriteAtStart(t *testing.T) {
	f := iRobotFrame(10)
	c := f.NewCaret()
	defer c.Close()

	prefix := "The Truth of the Matter\n(insofaras): "
	gotBuf := make([]byte, len(prefix)+len(iRobot))
	want := make([]byte, len(iRobot), len(prefix)+len(iRobot))
	copy(want, iRobot)

	for i := 0; i < len(prefix); i++ {
		x := prefix[len(prefix)-1-i]

		c.Seek(0, SeekSet)
		c.WriteByte(x)
		if err := checkInvariants(f); err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}

		want = want[:len(want)+1]
		copy(want[1:], want)
		want[0] = x

		if got := readAllText(gotBuf[:0], f); !bytes.Equal(got, want) {
			t.Fatalf("i=%d:\ngot  % x\nwant % x", i, got, want)
		}
	}
}

func TestSetMaxWidth(t *testing.T) {
	f := new(Frame)
	f.SetFace(toyFace{})
	want := ""
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		if i%20 == 5 {
			c := f.NewCaret()
			c.WriteString(iRobot)
			c.Close()
			want += iRobot
		}
		f.SetMaxWidth(fixed.I(rng.Intn(20)))
		if err := checkInvariants(f); err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}
		if err := testRead(f, want); err != nil {
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
					c.splitBox(false)
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

func TestDelete(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	gotBytesBuf := make([]byte, 0, len(iRobot))
	wantBytesBuf := make([]byte, 0, len(iRobot))

	for i := 0; i < 100; i++ {
		f := iRobotFrame(rng.Intn(len(iRobot) + 1))
		c := f.NewCaret()
		defer c.Close()
		wantBytes := append(wantBytesBuf[:0], iRobot...)
		for j := 0; j < 8; j++ {
			// Delete the text in the range [x, y).
			x, y := rngIntPair(rng, len(wantBytes)+1)
			got, want := 0, y-x
			if rng.Intn(2) == 0 {
				c.Seek(int64(x), SeekSet)
				got = c.Delete(Forwards, want)
			} else {
				c.Seek(int64(y), SeekSet)
				got = c.Delete(Backwards, want)
			}
			if err := checkInvariants(f); err != nil {
				t.Fatalf("i=%d, j=%d, x=%d, y=%d: %v", i, j, x, y, err)
			}
			if got != want {
				t.Fatalf("i=%d, j=%d, x=%d, y=%d: Delete: got %d, want %d", i, j, x, y, got, want)
			}

			gotBytes := readAllText(gotBytesBuf[:0], f)
			wantBytes = append(wantBytes[:x], wantBytes[y:]...)
			if !bytes.Equal(gotBytes, wantBytes) {
				t.Fatalf("i=%d, j=%d, x=%d, y=%d:\ngot  %q\nwant %q", i, j, x, y, gotBytes, wantBytes)
			}
		}
	}
}

func TestDeleteTooMuch(t *testing.T) {
	if len(iRobot) < 15+17 {
		t.Fatal("iRobot string is too short")
	}

	f := iRobotFrame(10)
	c := f.NewCaret()
	defer c.Close()

	c.Seek(-15, SeekEnd)
	if got, want := c.Delete(Forwards, 18), 15; got != want {
		t.Errorf("Delete Forwards: got %d, want %d", got, want)
	}

	c.Seek(+17, SeekSet)
	if got, want := c.Delete(Backwards, 18), 17; got != want {
		t.Errorf("Delete Backwards: got %d, want %d", got, want)
	}

	got := string(readAllText(nil, f))
	want := iRobot[17 : len(iRobot)-15]
	if got != want {
		t.Errorf("\ngot  %q\nwant %q", got, want)
	}
}

func TestMergeIntoOneLine(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		f := new(Frame)
		f.SetMaxWidth(fixed.I(100))

		if !f.initialized() {
			f.initialize()
		}
		p := f.firstP
		pp := &f.paragraphs[p]
		l := pp.firstL
		ll := &f.lines[l]
		b := ll.firstB
		bb := &f.boxes[b]

		// Make some Lines and Boxes.
		wantNUsedBoxes := 0
		prevIJ := int32(0)
		emptyRun := true
		for j := 0; ; j++ {
			length := rng.Intn(3)
			bb.i = prevIJ
			prevIJ += int32(length)
			bb.j = prevIJ
			f.len += length
			if length > 0 && emptyRun {
				emptyRun = false
				wantNUsedBoxes++
			}

			if rng.Intn(20) == 0 {
				break
			}

			if rng.Intn(4) == 0 {
				// Make a new Box on a new Line.
				l1, realloc := f.newLine()
				if realloc {
					ll = &f.lines[l]
				}
				ll1 := &f.lines[l1]
				ll.next = l1
				ll1.prev = l
				l, ll = l1, ll1

				ll.firstB, _ = f.newBox()
				b = ll.firstB
				bb = &f.boxes[b]

			} else {
				// Make a new Box on the same Line.
				b1, realloc := f.newBox()
				if realloc {
					bb = &f.boxes[b]
				}
				bb1 := &f.boxes[b1]
				bb.next = b1
				bb1.prev = b
				b, bb = b1, bb1
			}

			if rng.Intn(5) == 0 {
				// Put an i/j gap between this run and the previous one.
				prevIJ += 1 + int32(rng.Intn(5))
				emptyRun = true
			}
		}

		// We normally remove all empty Boxes. However, if there is no text
		// whatsoever, we still need one Box.
		if f.len == 0 {
			if wantNUsedBoxes != 0 {
				t.Fatalf("i=%d: no text: wantNUsedBoxes: got %d, want 0", i, wantNUsedBoxes)
			}
			wantNUsedBoxes = 1
		}

		// Make f.text long enough to hold all of the Boxes' i's and j's.
		f.text = make([]byte, prevIJ)
		for i := range f.text {
			f.text[i] = byte('a' + i%16)
		}

		// Do the merge.
		if err := checkSomeInvariants(f, ignoreInvariantEmptyBoxes); err != nil {
			t.Fatalf("i=%d: before: %v", i, err)
		}
		f.mergeIntoOneLine(p)
		if err := checkInvariants(f); err != nil {
			t.Fatalf("i=%d: after: %v", i, err)
		}

		// Check that there is only one Line in use.
		nUsedLines := 0
		for l, ll := range f.lines {
			// The 0th index is a special case. A negative prev means that the
			// Line is in the free-list.
			if l != 0 && ll.prev >= 0 {
				nUsedLines++
			}
		}
		if nUsedLines != 1 {
			t.Errorf("i=%d: nUsedLines: got %d, want %d", i, nUsedLines, 1)
		}

		// Check the number of Boxes in use.
		nUsedBoxes := 0
		for b, bb := range f.boxes {
			// The 0th index is a special case. A negative prev means that the
			// Box is in the free-list.
			if b != 0 && bb.prev >= 0 {
				nUsedBoxes++
			}
		}
		if nUsedBoxes != wantNUsedBoxes {
			t.Errorf("i=%d: nUsedBoxes: got %d, want %d", i, nUsedBoxes, wantNUsedBoxes)
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

const (
	ignoreInvariantEmptyBoxes = 1 << iota
)

func checkInvariants(f *Frame) error {
	return checkSomeInvariants(f, 0)
}

func checkSomeInvariants(f *Frame, ignoredInvariants uint32) error {
	const infinity = 1e6

	if !f.initialized() {
		if !reflect.DeepEqual(*f, Frame{}) {
			return fmt.Errorf("uninitialized Frame is not zero-valued")
		}
		return nil
	}

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

	// Check that only the last Box can be empty.
	if ignoredInvariants&ignoreInvariantEmptyBoxes == 0 {
		for p := f.firstP; p != 0; {
			nextP := f.paragraphs[p].next
			for l := f.paragraphs[p].firstL; l != 0; {
				nextL := f.lines[l].next
				for b := f.lines[l].firstB; b != 0; {
					nextB := f.boxes[b].next

					emptyBox := f.boxes[b].i == f.boxes[b].j
					lastBox := nextP == 0 && nextL == 0 && nextB == 0
					if emptyBox && !lastBox {
						return fmt.Errorf("boxes[%d] is empty, but isn't the last Box", b)
					}

					b = nextB
				}
				l = nextL
			}
			p = nextP
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

	// Check that each Caret's pos is in the Frame's 0:len range, the Caret's k
	// is in its Box's i:j range, and its Box b is in its Line l is in its
	// Paragraph p.
	for i, c := range f.carets {
		if c.pos < 0 || f.len < int(c.pos) {
			return fmt.Errorf("caret[%d]: pos %d outside range [0, %d]", i, c.pos, f.len)
		}

		if c.b < 1 || len(f.boxes) < int(c.b) {
			return fmt.Errorf("caret[%d]: c.b %d outside range [1, %d]", i, c.b, len(f.boxes))
		}
		bb := &f.boxes[c.b]
		if c.k < bb.i || bb.j < c.k {
			return fmt.Errorf("caret[%d]: c.k %d outside range [%d, %d]", i, c.k, bb.i, bb.j)
		}

		if c.l < 1 || len(f.lines) < int(c.l) {
			return fmt.Errorf("caret[%d]: c.l %d outside range [1, %d]", i, c.l, len(f.lines))
		}
		if !f.lines[c.l].contains(f, c.b) {
			return fmt.Errorf("caret[%d]: Line %d does not contain Box %d", i, c.l, c.b)
		}

		if c.p < 1 || len(f.paragraphs) < int(c.p) {
			return fmt.Errorf("caret[%d]: c.p %d outside range [1, %d]", i, c.p, len(f.paragraphs))
		}
		if !f.paragraphs[c.p].contains(f, c.l) {
			return fmt.Errorf("caret[%d]: Paragraph %d does not contain Line %d", i, c.p, c.l)
		}
	}

	return nil
}

func (p *Paragraph) contains(f *Frame, l int32) bool {
	for x := p.firstL; x != 0; x = f.lines[x].next {
		if x == l {
			return true
		}
	}
	return false
}

func (l *Line) contains(f *Frame, b int32) bool {
	for x := l.firstB; x != 0; x = f.boxes[x].next {
		if x == b {
			return true
		}
	}
	return false
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
