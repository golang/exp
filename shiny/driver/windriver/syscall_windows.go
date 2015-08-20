// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

package windriver

import "syscall"

type point struct {
	X int32
	Y int32
}

type msg struct {
	Hwnd    syscall.Handle
	Message uint32
	Wparam  uintptr
	Lparam  uintptr
	Time    uint32
	Pt      point
}

//sys	getMessage(msg *msg, hwnd syscall.Handle, msgfiltermin uint32, msgfiltermax uint32) (ret int32, err error) [failretval==-1] = user32.GetMessageW
//sys	translateMessage(msg *msg) (done bool) = user32.TranslateMessage
//sys	dispatchMessage(msg *msg) (ret int32) = user32.DispatchMessageW
