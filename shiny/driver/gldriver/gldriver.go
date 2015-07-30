// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gldriver provides an OpenGL driver for accessing a screen.
package gldriver

import (
	"image"

	"golang.org/x/exp/shiny/screen"
)

// Main is called by the program's main function to run the graphical
// application.
//
// It calls f on the Screen, possibly in a separate goroutine, as some OS-
// specific libraries require being on 'the main thread'. It returns when f
// returns.
func Main(f func(screen.Screen)) {
	if err := main(f); err != nil {
		f(errScreen{err})
	}
}

// errScreen is a screen.Screen.
type errScreen struct {
	err error
}

func (e errScreen) NewBuffer(size image.Point) (screen.Buffer, error) {
	return nil, e.err
}

func (e errScreen) NewTexture(size image.Point) (screen.Texture, error) {
	return nil, e.err
}

func (e errScreen) NewWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	return nil, e.err
}
