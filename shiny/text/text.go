// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package text lays out paragraphs of text.
//
// A body of text is laid out into a Frame: Frames contain Paragraphs (stacked
// vertically), Paragraphs contain Lines (stacked vertically), and Lines
// contain Boxes (stacked horizontally). Each Box holds a []byte slice of the
// text. For example, to simply print a Frame's text from start to finish:
//
//	var f *text.Frame = etc
//	for p := f.FirstParagraph(); p != nil; p = p.Next(f) {
//		for l := p.FirstLine(f); l != nil; l = l.Next(f) {
//			for b := l.FirstBox(f); b != nil; b = b.Next(f) {
//				fmt.Print(b.Text(f))
//			}
//		}
//	}
//
// A Frame's structure (the tree of Paragraphs, Lines and Boxes), and its
// []byte text, are not modified directly. Instead, a Frame's maximum width can
// be re-sized, and text can be added and removed via Carets (which implement
// standard io interfaces). For example, to add some words to the end of a
// frame:
//
//	var f *text.Frame = etc
//	c := f.NewCaret()
//	c.Seek(0, text.SeekEnd)
//	c.WriteString("Not with a bang but a whimper.\n")
//	c.Close()
//
// Either way, such modifications can cause re-layout, which can add or remove
// Paragraphs, Lines and Boxes. The underlying memory for such structs can be
// re-used, so pointer values, such as of type *Box, should not be held over
// such modifications.
package text

// These constants are equal to os.SEEK_SET, os.SEEK_CUR and os.SEEK_END,
// understood by the io.Seeker interface, and are provided so that users of
// this package don't have to explicitly import "os".
const (
	SeekSet int = 0
	SeekCur int = 1
	SeekEnd int = 2
)

// Frame holds Paragraphs of text.
//
// The zero value is a valid Frame of empty text, which contains one Paragraph,
// which contains one Line, which contains one Box.
type Frame struct {
	// These slices hold the Frame's Paragraphs, Lines and Boxes, indexed by
	// fields such as Paragraph.firstLine and Box.next.
	//
	// Their contents are not necessarily in layout order. Each slice is
	// obviously backed by an array, but a Frame's list of children
	// (Paragraphs) forms a doubly-linked list, not an array list, so that
	// insertion has lower algorithmic complexity. Similarly for a Paragraph's
	// list of children (Lines) and a Line's list of children (Boxes).
	//
	// The 0'th index into each slice is a special case.
	//
	// A zero firstFoo field means that the parent holds a single, implicit
	// (lazily allocated), empty-but-not-nil *Foo child. Every Frame contains
	// at least one Paragraph. Similarly, every Paragraph contains at least one
	// Line, and every Line contains at least one Box.
	//
	// A zero next or prev field means that there is no such sibling.
	paragraphs []Paragraph
	lines      []Line
	boxes      []Box

	firstParagraph int32

	// len is the total length of the Frame's current textual content, in
	// bytes. It can be smaller then len(text), since that []byte can contain
	// 'holes' of deleted content.
	//
	// Like the paragraphs, lines and boxes slice-typed fields above, the text
	// []byte does not necessarily hold the textual content in layout order.
	// Instead, it holds the content in edit (insertion) order, with occasional
	// compactions. Again, the algorithmic complexity of insertions matters.
	len  int
	text []byte
}

func (f *Frame) newParagraph() int32 {
	if len(f.paragraphs) == 0 {
		// The 1 is because the 0'th index is a special case.
		f.paragraphs = make([]Paragraph, 1, 16)
	}
	f.paragraphs = append(f.paragraphs, Paragraph{})
	return int32(len(f.paragraphs) - 1)
}

func (f *Frame) newLine() int32 {
	if len(f.lines) == 0 {
		// The 1 is because the 0'th index is a special case.
		f.lines = make([]Line, 1, 16)
	}
	f.lines = append(f.lines, Line{})
	return int32(len(f.lines) - 1)
}

func (f *Frame) newBox() int32 {
	if len(f.boxes) == 0 {
		// The 1 is because the 0'th index is a special case.
		f.boxes = make([]Box, 1, 16)
	}
	f.boxes = append(f.boxes, Box{})
	return int32(len(f.boxes) - 1)
}

// FirstParagraph returns the first paragraph of this frame.
func (f *Frame) FirstParagraph() *Paragraph {
	if f.firstParagraph == 0 {
		// 0 means that the first Paragraph is implicit (and not allocated
		// yet), so we make an explicit one, with a non-zero index.
		f.firstParagraph = f.newParagraph()
	}
	return &f.paragraphs[f.firstParagraph]
}

// Len returns the number of bytes in the Frame's text.
func (f *Frame) Len() int {
	return f.len
}

// NewCaret returns a new Caret at the start of this Frame.
func (f *Frame) NewCaret() *Caret {
	panic("TODO")
}

// TODO: be able to set a frame's max width, and font face.

// Paragraph holds Lines of text.
type Paragraph struct {
	firstLine, next, prev int32
}

// FirstLine returns the first Line of this Paragraph.
//
// f is the Frame that contains the Paragraph.
func (p *Paragraph) FirstLine(f *Frame) *Line {
	if p.firstLine == 0 {
		// 0 means that the first Line is implicit (and not allocated yet), so
		// we make an explicit one, with a non-zero index.
		p.firstLine = f.newLine()
	}
	return &f.lines[p.firstLine]
}

// Next returns the next Paragraph after this one in the Frame.
//
// f is the Frame that contains the Paragraph.
func (p *Paragraph) Next(f *Frame) *Paragraph {
	if p.next == 0 {
		return nil
	}
	return &f.paragraphs[p.next]
}

// Line holds Boxes of text.
type Line struct {
	firstBox, next, prev int32
}

// FirstBox returns the first Box of this Line.
//
// f is the Frame that contains the Line.
func (l *Line) FirstBox(f *Frame) *Box {
	if l.firstBox == 0 {
		// 0 means that the first Box is implicit (and not allocated yet), so
		// we make an explicit one, with a non-zero index.
		l.firstBox = f.newBox()
	}
	return &f.boxes[l.firstBox]
}

// Next returns the next Line after this one in the Paragraph.
//
// f is the Frame that contains the Line.
func (l *Line) Next(f *Frame) *Line {
	if l.next == 0 {
		return nil
	}
	return &f.lines[l.next]
}

// Box holds a contiguous run of text.
type Box struct {
	next, prev int32
	// Frame.text[i:j] holds this Box's text.
	i, j int32
}

// Next returns the next Box after this one in the Line.
//
// f is the Frame that contains the Box.
func (b *Box) Next(f *Frame) *Box {
	if b.next == 0 {
		return nil
	}
	return &f.boxes[b.next]
}

// Text returns the Box's text.
//
// f is the Frame that contains the Box.
func (b *Box) Text(f *Frame) []byte {
	return f.text[b.i:b.j:b.j]
}
