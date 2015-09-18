// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !darwin !386,!amd64 ios
// +build !linux android

package gldriver

import (
	"fmt"
	"runtime"

	"golang.org/x/exp/shiny/screen"
)

func newWindow(width, height int32) uintptr { return 0 }
func showWindow(id uintptr) uintptr         { return 0 }
func closeWindow(id uintptr)                {}
func drawLoop(w *windowImpl)                {}

func main(f func(screen.Screen)) error {
	return fmt.Errorf("gldriver: unsupported GOOS/GOARCH %s/%s", runtime.GOOS, runtime.GOARCH)
}
