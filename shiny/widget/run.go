// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

// TODO: how does RunWindow's caller inject or process events (whether general
// like lifecycle events or app-specific)? How does it stop the event loop when
// the app's work is done?

// TODO: how do widgets signal that they need repaint or relayout?

// TODO: propagate keyboard / mouse / touch events.

// RunWindow creates a new window for s, with the given widget tree, and runs
// its event loop.
func RunWindow(s screen.Screen, root NodeWrapper) error {
	var (
		buf screen.Buffer
		t   Theme
	)
	defer func() {
		if buf != nil {
			buf.Release()
		}
	}()

	w, err := s.NewWindow(nil)
	if err != nil {
		return err
	}
	defer w.Release()
	rootNode := root.WrappedNode()
	for {
		switch e := w.NextEvent().(type) {
		case lifecycle.Event:
			if e.To == lifecycle.StageDead {
				return nil
			}

		case paint.Event:
			if buf != nil {
				w.Upload(image.Point{}, buf, buf.Bounds())
			}
			w.Publish()

		case size.Event:
			if buf != nil {
				buf.Release()
			}
			var err error
			buf, err = s.NewBuffer(e.Size())
			if err != nil {
				return err
			}
			t.DPI = float64(e.PixelsPerPt) * unit.PointsPerInch
			rootNode.Measure(&t)
			rootNode.Rect = e.Bounds()
			rootNode.Layout(&t)
			rootNode.Paint(&t, buf.RGBA(), image.Point{})

		case error:
			return e
		}
	}
}
