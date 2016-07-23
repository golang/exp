// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"golang.org/x/exp/shiny/widget/node"
)

// Space is leaf widget that occupies empty space. For example, aligning two
// widgets to the left and right edges of a container can be achieved by
// placing a third Space widget between them, whose LayoutData makes that Space
// expand to occupy any excess space. Similarly, a widget can be centered in
// its container by adding an expanding Space before and after.
type Space struct {
	node.LeafEmbed
}

// NewSpace returns a new Space widget.
func NewSpace() *Space {
	w := &Space{}
	w.Wrapper = w
	return w
}
