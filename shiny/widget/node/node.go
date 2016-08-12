// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package node provides the structure for a tree of heterogenous widget nodes.
//
// Most programmers should not need to import this package, only the top-level
// widget package. Only those that write custom widgets need to explicitly
// refer to the Node, Embed and related types.
//
// The Node interface is usually implemented by struct types that embed one of
// LeafEmbed, ShellEmbed or ContainerEmbed (all of which themselves embed an
// Embed), providing default implementations of all of Node's methods.
//
// The split between an outer wrapper (Node) interface type and an inner
// wrappee (Embed) struct type enables heterogenous nodes, such as a buttons
// and labels, in a widget tree where every node contains common fields such as
// position, size and tree structure links (parent, siblings and children).
//
// In a traditional object-oriented type system, this might be represented by
// the Button and Label types both subclassing the Node type. Go does not have
// inheritance, so the outer / inner split is composed explicitly. For example,
// the concrete Button type is a struct type that embeds an XxxEmbed (such as
// LeafEmbed), and the NewButton function sets the inner Embed's Wrapper field
// to point back to the outer value.
//
// There are three layers here (Button embeds LeafEmbed embeds Embed) instead
// of two. The intermediate layer exists because there needs to be a place to
// provide default implementations of methods like Measure, but that place
// shouldn't be the inner-most type (Embed), otherwise it'd be too easy to
// write subtly incorrect code like:
//
//	for c := w.FirstChild; c != nil; c = c.NextSibling {
//		c.Measure(etc) // This should instead be c.Wrapper.Measure(etc).
//	}
//
// In any case, most programmers that want to construct a widget tree should
// not need to know this detail. It usually suffices to call functions such as
// widget.NewButton or widget.NewLabel, and then parent.Insert(button, nil).
//
// See the example/gallery program for some example code for a custom widget.
package node // import "golang.org/x/exp/shiny/widget/node"

import (
	"image"

	"golang.org/x/exp/shiny/gesture"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
)

// EventHandled is whether or not an input event (a key, mouse, touch or
// gesture event) was handled by a widget. If it was not handled, the event is
// propagated along the widget tree.
type EventHandled bool

const (
	NotHandled = EventHandled(false)
	Handled    = EventHandled(true)
)

// NoHint means that there is no width or height hint in a Measure call.
const NoHint = -1

// Node is a node in the widget tree.
type Node interface {
	// Wrappee returns the inner (embedded) type that is wrapped by this type.
	Wrappee() *Embed

	// Insert adds a node c as a child of this node. If nextSibling is nil, c
	// will be inserted at the end of this node's children. Otherwise, c will
	// be inserted such that its next sibling is nextSibling.
	//
	// It will panic if c already has a parent or siblings.
	Insert(c, nextSibling Node)

	// Remove removes a node c that is a child of this node. Afterwards, c will
	// have no parent and no siblings.
	//
	// It will panic if c's parent is not this node.
	Remove(c Node)

	// Measure sets this node's Embed.MeasuredSize to its natural size, taking
	// its children into account.
	//
	// Some nodes' natural height might depend on their imposed width, such as
	// a text widget word-wrapping its contents. The caller may provide hints
	// that the parent can override the child's natural size in the width,
	// height or both directions. A negative value means that there is no hint.
	// For example, a container might lay out its children to all have the same
	// width, and could pass that width as the widthHint argument.
	Measure(t *theme.Theme, widthHint, heightHint int)

	// Layout lays out this node (and its children), setting the Embed.Rect
	// fields of each child. This node's Embed.Rect field should have
	// previously been set during the parent node's layout.
	Layout(t *theme.Theme)

	// Paint paints this node (and its children). Painting is split into two
	// passes: a base pass and an effects pass. The effects pass is often a
	// no-op, and the bulk of the work is typically done in the base pass.
	//
	// The base pass paints onto an *image.RGBA pixel buffer and ancestor nodes
	// may choose to re-use the result. For example, re-painting a text widget
	// after scrolling may copy cached buffers at different offsets, instead of
	// painting the text's glyphs onto a fresh buffer. Similarly, animating the
	// scale and opacity of an overlay can re-use the buffer from a previous
	// base pass.
	//
	// The effects pass paints that part of the widget that can not or should
	// not be cached. For example, the border of a text widget shouldn't move
	// on the screen when that text widget is scrolled. The effects pass does
	// not have a destination RGBA pixel buffer, and is limited to what a
	// screen.Drawer provides: affine-transformed textures and uniform fills.
	//
	// TODO: app-specific OpenGL, if available, should be part of the effects
	// pass. Is that exposed via the screen.Drawer or by another mechanism?
	//
	// The Paint method may create base pass RGBA pixel buffers, by calling
	// ctx.Screen.NewBuffer. Many implementations won't, and instead assume
	// that PaintBase is recursively triggered by an ancestor node such as a
	// widget.Sheet. If it does create those RGBA pixel buffers, it is also
	// responsible for calling PaintBase on this node (and its children). In
	// any case, the Paint method should then paint any effects. Many widgets
	// will neither create their own buffers nor have any effects, so their
	// Paint methods will simply be the default implemention: do nothing except
	// call Paint on its children. As mentioned above, the bulk of the work is
	// typically done in PaintBase.
	//
	// origin is the parent widget's origin with respect to the ctx.Src2Dst
	// transformation matrix; this node's Embed.Rect.Add(origin) will be its
	// position and size in pre-transformed coordinate space.
	Paint(ctx *PaintContext, origin image.Point) error

	// PaintBase paints the base pass of this node (and its children) onto an
	// RGBA pixel buffer.
	//
	// origin is the parent widget's origin with respect to the ctx.Dst image's
	// origin; this node's Embed.Rect.Add(origin) will be its position and size
	// in ctx.Dst's coordinate space.
	PaintBase(ctx *PaintBaseContext, origin image.Point) error

	// Mark adds the given marks to this node. It calls OnChildMarked on its
	// parent if new marks were added.
	Mark(m Marks)

	// OnChildMarked handles a child being given new marks. By default, marks
	// are propagated up the node tree towards the root. For example, a child
	// being marked for needing paint will cause the parent being marked for
	// needing paint.
	OnChildMarked(child Node, newMarks Marks)

	// OnLifecycleEvent propagates a lifecycle event to a node (and its
	// children).
	OnLifecycleEvent(e lifecycle.Event)

	// OnInputEvent handles a key, mouse, touch or gesture event.
	//
	// origin is the parent widget's origin with respect to the event origin;
	// this node's Embed.Rect.Add(origin) will be its position and size in
	// event coordinate space.
	OnInputEvent(e interface{}, origin image.Point) EventHandled

	// TODO: other OnXxxEvent methods?

}

// PaintContext is the context for the Node.Paint method.
type PaintContext struct {
	Theme   *theme.Theme
	Screen  screen.Screen
	Drawer  screen.Drawer
	Src2Dst f64.Aff3

	// TODO: add a clip rectangle?

	// TODO: add the DrawContext from the lifecycle event?
}

// PaintBaseContext is the context for the Node.PaintBase method.
type PaintBaseContext struct {
	Theme *theme.Theme
	Dst   *image.RGBA

	// TODO: add a clip rectangle? Or rely on the RGBA.SubImage method to pass
	// smaller Dst images?
}

// LeafEmbed is designed to be embedded in struct types for nodes with no
// children.
type LeafEmbed struct{ Embed }

func (m *LeafEmbed) Insert(c, nextSibling Node) {
	panic("node: Insert called for a leaf parent")
}

func (m *LeafEmbed) Remove(c Node) { m.remove(c) }

func (m *LeafEmbed) Measure(t *theme.Theme, widthHint, heightHint int) { m.MeasuredSize = image.Point{} }

func (m *LeafEmbed) Layout(t *theme.Theme) {}

func (m *LeafEmbed) Paint(ctx *PaintContext, origin image.Point) error {
	m.Marks.UnmarkNeedsPaint()
	return nil
}

func (m *LeafEmbed) PaintBase(ctx *PaintBaseContext, origin image.Point) error {
	m.Marks.UnmarkNeedsPaintBase()
	return nil
}

func (m *LeafEmbed) OnChildMarked(child Node, newMarks Marks) {}

func (m *LeafEmbed) OnLifecycleEvent(e lifecycle.Event) {}

func (m *LeafEmbed) OnInputEvent(e interface{}, origin image.Point) EventHandled { return NotHandled }

// ShellEmbed is designed to be embedded in struct types for nodes with at most
// one child.
type ShellEmbed struct{ Embed }

func (m *ShellEmbed) Insert(c, nextSibling Node) {
	if m.FirstChild != nil {
		panic("node: Insert called for a shell parent that already has a child")
	}
	m.insert(c, nextSibling)
}

func (m *ShellEmbed) Remove(c Node) { m.remove(c) }

func (m *ShellEmbed) Measure(t *theme.Theme, widthHint, heightHint int) {
	if c := m.FirstChild; c != nil {
		c.Wrapper.Measure(t, widthHint, heightHint)
		m.MeasuredSize = c.MeasuredSize
	} else {
		m.MeasuredSize = image.Point{}
	}
}

func (m *ShellEmbed) Layout(t *theme.Theme) {
	if c := m.FirstChild; c != nil {
		c.Rect = m.Rect.Sub(m.Rect.Min)
		c.Wrapper.Layout(t)
	}
}

func (m *ShellEmbed) Paint(ctx *PaintContext, origin image.Point) error {
	m.Marks.UnmarkNeedsPaint()
	if c := m.FirstChild; c != nil {
		return c.Wrapper.Paint(ctx, origin.Add(m.Rect.Min))
	}
	return nil
}

func (m *ShellEmbed) PaintBase(ctx *PaintBaseContext, origin image.Point) error {
	m.Marks.UnmarkNeedsPaintBase()
	if c := m.FirstChild; c != nil {
		return c.Wrapper.PaintBase(ctx, origin.Add(m.Rect.Min))
	}
	return nil
}

func (m *ShellEmbed) OnChildMarked(child Node, newMarks Marks) {
	m.Mark(newMarks)
}

func (m *ShellEmbed) OnLifecycleEvent(e lifecycle.Event) {
	if c := m.FirstChild; c != nil {
		c.Wrapper.OnLifecycleEvent(e)
	}
}

func (m *ShellEmbed) OnInputEvent(e interface{}, origin image.Point) EventHandled {
	if c := m.FirstChild; c != nil {
		return c.Wrapper.OnInputEvent(e, origin.Add(m.Rect.Min))
	}
	return NotHandled
}

// ContainerEmbed is designed to be embedded in struct types for nodes with any
// number of children.
type ContainerEmbed struct{ Embed }

func (m *ContainerEmbed) Insert(c, nextSibling Node) { m.insert(c, nextSibling) }

func (m *ContainerEmbed) Remove(c Node) { m.remove(c) }

func (m *ContainerEmbed) Measure(t *theme.Theme, widthHint, heightHint int) {
	mSize := image.Point{}
	for c := m.FirstChild; c != nil; c = c.NextSibling {
		c.Wrapper.Measure(t, NoHint, NoHint)
		if mSize.X < c.MeasuredSize.X {
			mSize.X = c.MeasuredSize.X
		}
		if mSize.Y < c.MeasuredSize.Y {
			mSize.Y = c.MeasuredSize.Y
		}
	}
	m.MeasuredSize = mSize
}

func (m *ContainerEmbed) Layout(t *theme.Theme) {
	for c := m.FirstChild; c != nil; c = c.NextSibling {
		c.Rect = image.Rectangle{Max: c.MeasuredSize}
		c.Wrapper.Layout(t)
	}
}

func (m *ContainerEmbed) Paint(ctx *PaintContext, origin image.Point) error {
	m.Marks.UnmarkNeedsPaint()
	origin = origin.Add(m.Rect.Min)
	for c := m.FirstChild; c != nil; c = c.NextSibling {
		if err := c.Wrapper.Paint(ctx, origin); err != nil {
			return err
		}
	}
	return nil
}

func (m *ContainerEmbed) PaintBase(ctx *PaintBaseContext, origin image.Point) error {
	m.Marks.UnmarkNeedsPaintBase()
	origin = origin.Add(m.Rect.Min)
	for c := m.FirstChild; c != nil; c = c.NextSibling {
		if err := c.Wrapper.PaintBase(ctx, origin); err != nil {
			return err
		}
	}
	return nil
}

func (m *ContainerEmbed) OnChildMarked(child Node, newMarks Marks) {
	m.Mark(newMarks)
}

func (m *ContainerEmbed) OnLifecycleEvent(e lifecycle.Event) {
	for c := m.FirstChild; c != nil; c = c.NextSibling {
		c.Wrapper.OnLifecycleEvent(e)
	}
}

func (m *ContainerEmbed) OnInputEvent(e interface{}, origin image.Point) EventHandled {
	origin = origin.Add(m.Rect.Min)
	var p image.Point
	switch e := e.(type) {
	case gesture.Event:
		p = image.Point{
			X: int(e.CurrentPos.X) - origin.X,
			Y: int(e.CurrentPos.Y) - origin.Y,
		}
	case mouse.Event:
		p = image.Point{
			X: int(e.X) - origin.X,
			Y: int(e.Y) - origin.Y,
		}
	}
	// Iterate backwards. Later children have priority over earlier children,
	// as later ones are usually drawn over earlier ones.
	for c := m.LastChild; c != nil; c = c.PrevSibling {
		if p.In(c.Rect) && c.Wrapper.OnInputEvent(e, origin) == Handled {
			return Handled
		}
	}
	return NotHandled
}

// Embed is the common data structure for each node in a widget tree.
type Embed struct {
	// Wrapper is the outer type that wraps (embeds) this type. It should not
	// be nil.
	Wrapper Node

	// Parent, FirstChild, LastChild, PrevSibling and NextSibling describe the
	// widget tree structure.
	//
	// These fields are exported to enable walking the node tree, but they
	// should not be modified directly. Instead, call the Insert and Remove
	// methods, which keeps the tree structure consistent.
	Parent, FirstChild, LastChild, PrevSibling, NextSibling *Embed

	// LayoutData is layout-specific data for this node. Its type is determined
	// by its parent node's type. For example, each child of a Flow may hold a
	// FlowLayoutData in this field.
	LayoutData interface{}

	// TODO: add commentary about the Measure / Layout / Paint model, and about
	// the lifetime of the MeasuredSize and Rect fields, and when user code can
	// access and/or modify them. At some point a new cycle begins, a call to
	// measure is necessary, and using MeasuredSize is incorrect (unless you're
	// trying to recall something about the past).

	// MeasuredSize is the widget's natural size, in pixels, as calculated by
	// the most recent Measure call.
	MeasuredSize image.Point

	// Rect is the widget's position and actual (as opposed to natural) size,
	// in pixels, as calculated by the most recent Layout call on its parent
	// node. A parent may lay out a child at a size different to its natural
	// size in order to satisfy a layout constraint, such as a row of buttons
	// expanding to fill a panel's width.
	//
	// The position (Rectangle.Min) is relative to its parent node. This is not
	// necessarily the same as relative to the screen's, window's or image
	// buffer's origin.
	Rect image.Rectangle

	// Marks are a bitfield of node state, such as whether it needs measure,
	// layout or paint.
	Marks Marks
}

func (m *Embed) Wrappee() *Embed { return m }

// TODO: should insert and remove call Mark(MarkNeedsMeasureLayout | MarkNeedsPaint)?

func (m *Embed) insert(c, nextSibling Node) {
	n := c.Wrappee()
	if n.Parent != nil || n.PrevSibling != nil || n.NextSibling != nil {
		panic("node: Insert called for an attached child")
	}
	n.Parent = m

	if nextSibling == nil {
		last := m.LastChild
		if last != nil {
			last.NextSibling = n
		} else {
			m.FirstChild = n
		}
		m.LastChild = n
		n.PrevSibling = last
		return
	}

	o := nextSibling.Wrappee()
	if o.Parent != m {
		panic("node: Insert called for a non-sibling nextSibling node")
	}
	if o.PrevSibling == nil {
		o.PrevSibling = n
		n.NextSibling = o
		m.FirstChild = n
		return
	}

	o.PrevSibling.NextSibling = n
	n.PrevSibling = o.PrevSibling
	n.NextSibling = o
	o.PrevSibling = n
}

func (m *Embed) remove(c Node) {
	n := c.Wrappee()
	if n.Parent != m {
		panic("node: Remove called for a non-child node")
	}
	if m.FirstChild == n {
		m.FirstChild = n.NextSibling
	}
	if n.NextSibling != nil {
		n.NextSibling.PrevSibling = n.PrevSibling
	}
	if m.LastChild == n {
		m.LastChild = n.PrevSibling
	}
	if n.PrevSibling != nil {
		n.PrevSibling.NextSibling = n.NextSibling
	}
	n.Parent = nil
	n.PrevSibling = nil
	n.NextSibling = nil
}

func (m *Embed) Mark(marks Marks) {
	oldMarks := m.Marks
	m.Marks |= marks
	changedMarks := m.Marks ^ oldMarks
	if changedMarks != 0 && m.Parent != nil {
		m.Parent.Wrapper.OnChildMarked(m.Wrapper, changedMarks)
	}
}

// Marks are a bitfield of node state, such as whether it needs measure, layout
// or paint.
type Marks uint32

const (
	// MarkNeedsMeasureLayout marks this node as needing Measure and Layout
	// calls.
	MarkNeedsMeasureLayout = Marks(1 << 0)
	// TODO: use this.

	// MarkNeedsPaint marks this node as needing a Paint call.
	MarkNeedsPaint = Marks(1 << 1)

	// MarkNeedsPaintBase marks this node as needing a PaintBase call.
	MarkNeedsPaintBase = Marks(1 << 2)
)

func (m Marks) NeedsMeasureLayout() bool { return m&MarkNeedsMeasureLayout != 0 }
func (m Marks) NeedsPaint() bool         { return m&MarkNeedsPaint != 0 }
func (m Marks) NeedsPaintBase() bool     { return m&MarkNeedsPaintBase != 0 }

func (m *Marks) UnmarkNeedsMeasureLayout() { *m &^= MarkNeedsMeasureLayout }
func (m *Marks) UnmarkNeedsPaint()         { *m &^= MarkNeedsPaint }
func (m *Marks) UnmarkNeedsPaintBase()     { *m &^= MarkNeedsPaintBase }
