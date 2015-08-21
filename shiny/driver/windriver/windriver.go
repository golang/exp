// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

// #include "windriver.h"
import "C"

import (
	"runtime"

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

	hr := C.initUtilityWindow()
	if hr != C.S_OK {
		return winerror("failed to create utility window", hr)
	}
	defer func() {
		// TODO(andlabs): log an error if this fails?
		C.DestroyWindow(C.utilityWindow)
		// TODO(andlabs): unregister window class
	}()

	hr = C.initWindowClass()
	if hr != C.S_OK {
		return winerror("failed to create Window window class", hr)
	}
	// TODO(andlabs): uninit

	s := newScreenImpl()
	go f(s)

	mainMessagePump()
	return nil
}

func mainMessagePump() {
	var m msg
	for {
		// This GetMessage cannot fail: http://blogs.msdn.com/b/oldnewthing/archive/2013/03/22/10404367.aspx
		// TODO(andlabs): besides, what should we do if a future Windows change makes it fail for some other reason? we can't return an error because it's too late to stop the main function
		_, err := getMessage(&m, 0, 0, 0)
		if err != nil {
			return
		}
		translateMessage(&m)
		dispatchMessage(&m)
	}
}
