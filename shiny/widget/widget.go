// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package widget provides graphical user interface widgets.
//
// TODO: give an overview and some example code.
package widget // import "golang.org/x/exp/shiny/widget"

import (
	"image"

	"golang.org/x/exp/shiny/gesture"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
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

func (a Axis) Horizontal() bool { return a&AxisHorizontal != 0 }
func (a Axis) Vertical() bool   { return a&AxisVertical != 0 }

// WithLayoutData returns the given node after setting its embedded LayoutData
// field.
func WithLayoutData(n node.Node, layoutData interface{}) node.Node {
	n.Wrappee().LayoutData = layoutData
	return n
}

// RunWindowOptions are optional arguments to RunWindow.
type RunWindowOptions struct {
	NewWindowOptions screen.NewWindowOptions
	Theme            theme.Theme

	// TODO: some mechanism to process, filter and inject events. Perhaps a
	// screen.EventFilter interface, and note that the zero value in this
	// RunWindowOptions implicitly includes the gesture.EventFilter?
}

// TODO: how does RunWindow's caller inject or process events (whether general
// like lifecycle events or app-specific)? How does it stop the event loop when
// the app's work is done?

// TODO: how do widgets signal that they need repaint or relayout?

// TODO: propagate keyboard / mouse / touch events.

// RunWindow creates a new window for s, with the given widget tree, and runs
// its event loop.
//
// A nil opts is valid and means to use the default option values.
func RunWindow(s screen.Screen, root node.Node, opts *RunWindowOptions) error {
	var (
		nwo *screen.NewWindowOptions
		t   *theme.Theme
	)
	if opts != nil {
		nwo = &opts.NewWindowOptions
		t = &opts.Theme
	}
	w, err := s.NewWindow(nwo)
	if err != nil {
		return err
	}
	defer w.Release()

	// paintPending batches up multiple NeedsPaint observations so that we
	// paint only once (which can be relatively expensive) even when there are
	// multiple input events in the queue, such as from a rapidly moving mouse
	// or from the user typing many keys.
	//
	// TODO: determine somehow if there's an external paint event in the queue,
	// not just internal paint events?
	//
	// TODO: if every package that uses package screen should basically
	// throttle like this, should it be provided at a lower level?
	paintPending := false

	gef := gesture.EventFilter{EventDeque: w}
	for {
		e := w.NextEvent()

		if e = gef.Filter(e); e == nil {
			continue
		}

		switch e := e.(type) {
		case lifecycle.Event:
			root.OnLifecycleEvent(e)
			if e.To == lifecycle.StageDead {
				return nil
			}

		case gesture.Event, mouse.Event:
			root.OnInputEvent(e, image.Point{})

		case paint.Event:
			ctx := &node.PaintContext{
				Theme:  t,
				Screen: s,
				Drawer: w,
				Src2Dst: f64.Aff3{
					1, 0, 0,
					0, 1, 0,
				},
			}
			if err := root.Paint(ctx, image.Point{}); err != nil {
				return err
			}
			w.Publish()
			paintPending = false

		case size.Event:
			if dpi := float64(e.PixelsPerPt) * unit.PointsPerInch; dpi != t.GetDPI() {
				newT := new(theme.Theme)
				if t != nil {
					*newT = *t
				}
				newT.DPI = dpi
				t = newT
			}

			size := e.Size()
			root.Measure(t, size.X, size.Y)
			root.Wrappee().Rect = e.Bounds()
			root.Layout(t)
			// TODO: call Mark(node.MarkNeedsPaint)?

		case error:
			return e
		}

		if !paintPending && root.Wrappee().Marks.NeedsPaint() {
			paintPending = true
			w.Send(paint.Event{})
		}
	}
}
