// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package widget provides graphical user interface widgets.
//
// TODO: give an overview and some example code.
package widget // import "golang.org/x/exp/shiny/widget"

import (
	"image"
)

// Arity is the number of children a class of nodes can have.
type Arity uint8

const (
	Leaf      = Arity(0) // Leaf nodes have no children.
	Shell     = Arity(1) // Shell nodes have at most one child.
	Container = Arity(2) // Container nodes can have any number of children.
)

// Axis is zero, one or both of the horizontal and vertical axes. For example,
// a widget may be scrollable in one of the four AxisXxx values.
type Axis uint8

const (
	AxisNone       = Axis(0)
	AxisHorizontal = Axis(1)
	AxisVertical   = Axis(2)
	AxisBoth       = Axis(3) // AxisBoth equals AxisHorizontal | AxisVertical.
)

// Class is a class of nodes. For example, all button widgets would be Nodes
// whose Class values are a ButtonClass.
type Class interface {
	// Arity returns the number of children this class of nodes can have.
	Arity() Arity

	// Measure sets n.MeasuredSize to the natural size, in pixels, of a
	// specific node (and its children) of this class.
	Measure(n *Node, t *Theme)

	// Layout lays out a specific node (and its children) of this class,
	// setting the Node.Rect fields of each child. The n.Rect field should have
	// previously been set during the parent node's layout.
	Layout(n *Node, t *Theme)

	// Paint paints a specific node (and its children) of this class onto a
	// destination image.
	Paint(n *Node, t *Theme, dst *image.RGBA)

	// TODO: OnXxxEvent methods.
}

// LeafClassEmbed is designed to be embedded in struct types that implement the
// Class interface and have Leaf arity. It provides default implementations of
// the Class interface's methods.
type LeafClassEmbed struct{}

func (LeafClassEmbed) Arity() Arity                             { return Leaf }
func (LeafClassEmbed) Measure(n *Node, t *Theme)                { n.MeasuredSize = image.Point{} }
func (LeafClassEmbed) Layout(n *Node, t *Theme)                 {}
func (LeafClassEmbed) Paint(n *Node, t *Theme, dst *image.RGBA) {}

// ShellClassEmbed is designed to be embedded in struct types that implement
// the Class interface and have Shell arity. It provides default
// implementations of the Class interface's methods.
type ShellClassEmbed struct{}

func (ShellClassEmbed) Arity() Arity { return Shell }

func (ShellClassEmbed) Measure(n *Node, t *Theme) {
	if c := n.FirstChild; c != nil {
		c.Class.Measure(c, t)
		n.MeasuredSize = c.MeasuredSize
	} else {
		n.MeasuredSize = image.Point{}
	}
}

func (ShellClassEmbed) Layout(n *Node, t *Theme) {
	if c := n.FirstChild; c != nil {
		c.Rect = n.Rect
		c.Class.Layout(c, t)
	}
}

func (ShellClassEmbed) Paint(n *Node, t *Theme, dst *image.RGBA) {
	if c := n.FirstChild; c != nil {
		c.Class.Paint(c, t, dst)
	}
}

// ContainerClassEmbed is designed to be embedded in struct types that
// implement the Class interface and have Container arity. It provides default
// implementations of the Class interface's methods.
type ContainerClassEmbed struct{}

func (ContainerClassEmbed) Arity() Arity { return Container }

func (ContainerClassEmbed) Measure(n *Node, t *Theme) {
	mSize := image.Point{}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		c.Class.Measure(c, t)
		if mSize.X < c.MeasuredSize.X {
			mSize.X = c.MeasuredSize.X
		}
		if mSize.Y < c.MeasuredSize.Y {
			mSize.Y = c.MeasuredSize.Y
		}
	}
	n.MeasuredSize = mSize
}

func (ContainerClassEmbed) Layout(n *Node, t *Theme) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		c.Rect = image.Rectangle{Max: c.MeasuredSize}
		c.Class.Layout(c, t)
	}
}

func (ContainerClassEmbed) Paint(n *Node, t *Theme, dst *image.RGBA) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		c.Class.Paint(c, t, dst)
	}
}

// Node is an element of a widget tree.
//
// Every element of a widget tree is a node, but nodes can be of different
// classes. For example, a Flow node (i.e. one whose Class is FlowClass) can
// contain two Button nodes and an Image node.
type Node struct {
	// Parent, FirstChild, LastChild, PrevSibling and NextSibling describe the
	// widget tree structure.
	Parent, FirstChild, LastChild, PrevSibling, NextSibling *Node

	// Class is what class of node this is.
	Class Class

	// ClassData is class-specific data for this node. For example, a
	// ButtonClass may store an image and some text in this field. A
	// ProgressBarClass may store a numerical percentage.
	ClassData interface{}

	// TODO: add commentary about the Measure / Layout / Paint model, and about
	// the lifetime of the MeasuredSize and Rect fields, and when user code can
	// access and/or modify them. At some point a new cycle begins, a call to
	// measure is necessary, and using MeasuredSize is incorrect (unless you're
	// trying to recall something about the past).

	// MeasuredSize is the widget's natural size, in pixels, as calculated by
	// the most recent Class.Measure call.
	MeasuredSize image.Point

	// Rect is the widget's position and actual (as opposed to natural) size,
	// in pixels, as calculated by the most recent Class.Layout call on its
	// parent node. A parent may lay out a child at a size different to its
	// natural size in order to satisfy a layout constraint, such as a row of
	// buttons expanding to fill a panel's width.
	//
	// The position (Rectangle.Min) is relative to its parent node. This is not
	// necessarily the same as relative to the screen's, window's or image
	// buffer's origin.
	Rect image.Rectangle
}

// AppendChild adds a node c as a child of n.
//
// It will panic if c already has a parent or siblings.
func (n *Node) AppendChild(c *Node) {
	if c.Parent != nil || c.PrevSibling != nil || c.NextSibling != nil {
		panic("widget: AppendChild called for an attached child Node")
	}
	switch n.Class.Arity() {
	case Leaf:
		panic("widget: AppendChild called for a leaf parent Node")
	case Shell:
		if n.FirstChild != nil {
			panic("widget: AppendChild called for a shell parent Node that already has a child Node")
		}
	}
	last := n.LastChild
	if last != nil {
		last.NextSibling = c
	} else {
		n.FirstChild = c
	}
	n.LastChild = c
	c.Parent = n
	c.PrevSibling = last
}

// RemoveChild removes a node c that is a child of n. Afterwards, c will have
// no parent and no siblings.
//
// It will panic if c's parent is not n.
func (n *Node) RemoveChild(c *Node) {
	if c.Parent != n {
		panic("widget: RemoveChild called for a non-child Node")
	}
	if n.FirstChild == c {
		n.FirstChild = c.NextSibling
	}
	if c.NextSibling != nil {
		c.NextSibling.PrevSibling = c.PrevSibling
	}
	if n.LastChild == c {
		n.LastChild = c.PrevSibling
	}
	if c.PrevSibling != nil {
		c.PrevSibling.NextSibling = c.NextSibling
	}
	c.Parent = nil
	c.PrevSibling = nil
	c.NextSibling = nil
}
