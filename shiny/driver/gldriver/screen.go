// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build 386 amd64

package gldriver

// #include "cocoa.h"
import "C"

import (
	"fmt"
	"image"
	"sync"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/paint"
)

type screenImpl struct {
	mu      sync.Mutex
	windows map[uintptr]*windowImpl
}

func (s *screenImpl) NewBuffer(size image.Point) (retBuf screen.Buffer, retErr error) {
	return &bufferImpl{
		rgba: image.NewRGBA(image.Rectangle{Max: size}),
		size: size,
	}, nil
}

func (s *screenImpl) NewTexture(size image.Point) (screen.Texture, error) {
	return nil, fmt.Errorf("NewTexture not implemented")
}

func (s *screenImpl) NewWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	// TODO: look at opts.
	const width, height = 512, 384

	id := C.newWindow(width, height)
	w := &windowImpl{
		s:         s,
		id:        uintptr(id),
		eventsIn:  make(chan interface{}),
		eventsOut: make(chan interface{}),
		endPaint:  make(chan paint.Event, 1),
		draw:      make(chan struct{}),
		drawDone:  make(chan struct{}),
	}

	s.mu.Lock()
	s.windows[uintptr(id)] = w
	s.mu.Unlock()

	go w.pump()
	go w.drawLoop(uintptr(C.showWindow(id)))

	return w, nil
}
