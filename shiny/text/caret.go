// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

// TODO: do we care about "\n" vs "\r" vs "\r\n"? We only recognize "\n" for
// now.

import (
	"strings"
)

// Caret is a location in a Frame's text, and is the mechanism for adding and
// removing bytes of text. Conceptually, a Caret and a Frame's text is like an
// int c and a []byte t such that the text before and after that Caret is t[:c]
// and t[c:]. That byte-count location remains unchanged even when a Frame is
// re-sized and laid out into a new tree of Paragraphs, Lines and Boxes.
//
// A Frame can have multiple open Carets. For example, the beginning and end of
// a text selection can be represented by two Carets. Multiple Carets for the
// one Frame are not safe to use concurrently, but it is valid to interleave
// such operations sequentially. For example, if two Carets c0 and c1 for the
// one Frame are positioned at the 10th and 20th byte, and 4 bytes are written
// to c0, inserting what becomes the equivalent of text[10:14], then c0's
// position is updated to be 14 but c1's position is also updated to be 24.
type Caret struct {
	f *Frame

	// caretsIndex is the index of this Caret in the f.carets slice.
	caretsIndex int

	// p, l and b index the Caret's Paragraph, Line and Box. None of these
	// values can be zero.
	p, l, b int32

	// pos is the Caret's position in the text, in layout order. It is the "c"
	// as in "t[:c]" in the doc comment for type Caret above. It is not valid
	// to index the Frame.text slice with pos, since the Frame.text slice does
	// not necessarily hold the textual content in layout order.
	pos int32

	// k is the Caret's position in the text, in Frame.text order. It is valid
	// to index the Frame.text slice with k, analogous to the Box.i and Box.j
	// fields. For a Caret c, and letting bb := c.f.boxes[c.b], an invariant is
	// that bb.i <= c.k && c.k <= bb.j.
	k int32
}

// Close closes the Caret.
func (c *Caret) Close() error {
	i, j := c.caretsIndex, len(c.f.carets)-1

	// Swap c with the last element of c.f.carets.
	if i != j {
		other := c.f.carets[j]
		other.caretsIndex = i
		c.f.carets[i] = other
	}

	c.f.carets[j] = nil
	c.f.carets = c.f.carets[:j]
	*c = Caret{}
	return nil
}

// WriteString inserts s into the Frame's text at the Caret.
//
// The error returned is always nil.
func (c *Caret) WriteString(s string) (n int, err error) {
	n = len(s)
	for len(s) > 0 {
		i := 1 + strings.IndexByte(s, '\n')
		if i == 0 {
			i = len(s)
		}
		c.writeString(s[:i])
		s = s[i:]
	}
	return n, nil
}

// writeString inserts s into the Frame's text at the Caret.
//
// s must be non-empty, it must contain at most one '\n' and if it does contain
// one, it must be the final byte.
func (c *Caret) writeString(s string) {
	// If the Box's text is empty, move its empty i:j range to the equivalent
	// empty range at the end of c.f.text.
	if bb, n := &c.f.boxes[c.b], int32(len(c.f.text)); bb.i == bb.j && bb.i != n {
		bb.i = n
		bb.j = n
		for _, cc := range c.f.carets {
			if cc.b == c.b {
				cc.k = n
			}
		}
	}

	if c.k != int32(len(c.f.text)) {
		panic("TODO: inserting text somewhere other than at the end of the text buffer")
	}

	// Assert that the Caret c is at the end of its Box, and that Box's text is
	// at the end of the Frame's buffer.
	if c.k != c.f.boxes[c.b].j || c.k != int32(len(c.f.text)) {
		panic("text: invalid state")
	}

	c.f.text = append(c.f.text, s...)
	c.f.len += len(s)
	c.f.boxes[c.b].j += int32(len(s))
	c.k += int32(len(s))
	for _, cc := range c.f.carets {
		if cc.pos > c.pos {
			cc.pos += int32(len(s))
		}
	}
	c.pos += int32(len(s))
	oldP := c.p

	if s[len(s)-1] == '\n' {
		breakParagraph(c.f, c.p, c.l, c.b)
		c.p = c.f.paragraphs[c.p].next
		c.l = c.f.paragraphs[c.p].firstL
		c.b = c.f.lines[c.l].firstB
		c.k = c.f.boxes[c.b].i
	}

	// TODO: re-layout the oldP and c.p paragraphs.
	_ = oldP
}

// breakParagraph breaks the Paragraph p into two Paragraphs, just after Box b
// in Line l in Paragraph p. b's text must end with a '\n'. The new Paragraph
// is inserted after p.
func breakParagraph(f *Frame, p, l, b int32) {
	// Assert that the Box b's text ends with a '\n'.
	if j := f.boxes[b].j; j == 0 || f.text[j-1] != '\n' {
		panic("text: invalid state")
	}

	// Make a new, empty Paragraph after this Paragraph p.
	newP := f.newParagraph()
	nextP := f.paragraphs[p].next
	if nextP != 0 {
		f.paragraphs[nextP].prev = newP
	}
	f.paragraphs[newP].next = nextP
	f.paragraphs[newP].prev = p
	f.paragraphs[p].next = newP

	// Any Lines in this Paragraph after the break point's Line l move to the
	// newP Paragraph.
	if nextL := f.lines[l].next; nextL != 0 {
		f.lines[l].next = 0
		f.lines[nextL].prev = 0
		f.paragraphs[newP].firstL = nextL
	}

	// Any Boxes in this Line after the break point's Box b move to a new Line
	// at the start of the newP Paragraph.
	if nextB := f.boxes[b].next; nextB != 0 {
		f.boxes[b].next = 0
		f.boxes[nextB].prev = 0
		newL := f.newLine()
		f.lines[newL].firstB = nextB
		if newPFirstL := f.paragraphs[newP].firstL; newPFirstL != 0 {
			f.lines[newL].next = newPFirstL
			f.lines[newPFirstL].prev = newL
		}
		f.paragraphs[newP].firstL = newL
	}

	// Make the newP Paragraph's first Line and first Box explicit, since
	// Carets require an explicit p, l and b.
	{
		firstL := f.paragraphs[newP].firstLine(f)
		f.lines[firstL].firstBox(f)
	}

	// TODO: fix up other Carets's p, l and b fields.
	// TODO: re-layout the newP paragraph.
}

// TODO: many Caret methods.
