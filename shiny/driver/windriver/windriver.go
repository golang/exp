// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

// #cgo LDFLAGS: -lgdi32 -lmsimg32
// #include "windriver.h"
import "C"

import (
	"runtime"
	"syscall"

	"golang.org/x/exp/shiny/driver/internal/errscreen"
	"golang.org/x/exp/shiny/screen"
)

// TODO(andlabs): Should the Windows API code be split into a
// separate package internal/winbackend so gldriver can use it too?

// Main is called by the program's main function to run the graphical
// application.
//
// It calls f on the Screen, possibly in a separate goroutine, as some OS-
// specific libraries require being on 'the main thread'. It returns when f
// returns.
func Main(f func(screen.Screen)) {
	if err := main(f); err != nil {
		f(errscreen.Stub(err))
	}
}

func main(f func(screen.Screen)) (retErr error) {
	// It does not matter which OS thread we are on.
	// All that matters is that we confine all UI operations
	// to the thread that created the respective window.
	runtime.LockOSThread()

	if err := initCommon(); err != nil {
		return err
	}

	if err := initScreenWindow(); err != nil {
		return err
	}
	defer func() {
		// TODO(andlabs): log an error if this fails?
		_DestroyWindow(screenHWND)
		// TODO(andlabs): unregister window class
	}()

	if hr := C.initWindowClass(); hr != C.S_OK {
		return winerror("failed to create Window window class", hr)
	}
	// TODO(andlabs): uninit

	s := newScreenImpl()
	go f(s)

	mainMessagePump()
	return nil
}

var (
	hDefaultIcon   syscall.Handle
	hDefaultCursor syscall.Handle
	hThisInstance  syscall.Handle
)

func initCommon() (err error) {
	hDefaultIcon, err = _LoadIcon(0, _IDI_APPLICATION)
	if err != nil {
		return err
	}
	hDefaultCursor, err = _LoadCursor(0, _IDC_ARROW)
	if err != nil {
		return err
	}
	// TODO(andlabs) hThisInstance
	return nil
}

func mainMessagePump() {
	var m _MSG
	for {
		done, err := _GetMessage(&m, 0, 0, 0)
		if err != nil {
			// TODO
		}
		if done == 0 { // WM_QUIT
			return
		}
		_TranslateMessage(&m)
		_DispatchMessage(&m)
	}
}
