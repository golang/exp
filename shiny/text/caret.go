// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

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
	// TODO: implement.
}

// TODO: many Caret methods.
