// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !darwin
// +build !linux android

package driver

import (
	"errors"
	"image"

	"golang.org/x/exp/shiny/screen"
)

func main(f func(screen.Screen)) {
	f(stub{})
}

type stub struct{}

func (stub) NewBuffer(size image.Point) (screen.Buffer, error) {
	return nil, errNoDriver
}

func (stub) NewTexture(size image.Point) (screen.Texture, error) {
	return nil, errNoDriver
}

func (stub) NewWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	return nil, errNoDriver
}

var errNoDriver = errors.New("no driver for accessing a screen")
