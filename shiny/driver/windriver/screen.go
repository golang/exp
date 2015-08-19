// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

import (
	"fmt"
	"image"

	"golang.org/x/exp/shiny/screen"
)

type screenimpl struct{}

func newScreenImpl() screen.Screen {
	return &screenimpl{}
}

func (*screenimpl) NewBuffer(size image.Point) (screen.Buffer, error) {
	return nil, fmt.Errorf("TODO")
}

func (*screenimpl) NewTexture(size image.Point) (screen.Texture, error) {
	return nil, fmt.Errorf("TODO")
}

func (*screenimpl) NewWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	return newWindow(opts)
}
