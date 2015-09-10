// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"

	"golang.org/x/exp/shiny/screen"
)

// screenHWND is the handle to the "Screen window".
// The Screen window encapsulates all screen.Screen operations
// in an actual Windows window so they all run on the main thread.
// Since any messages sent to a window will be executed on the
// main thread, we can safely use the messages below.
var screenHWND syscall.Handle

const (
	// wParam - pointer to window options
	// lParam - pointer to *screen.Window
	// lResult - pointer to error
	msgCreateWindow = _WM_USER + iota
	msgFillSrc
	msgFillOver
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

type newWindowParams struct {
	opts *screen.NewWindowOptions
	w    screen.Window
	err  error
}

func (*screenimpl) NewWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	var p newWindowParams
	p.opts = opts
	_SendMessage(screenHWND, msgCreateWindow,
		0,
		uintptr(unsafe.Pointer(&p)))
	return p.w, p.err
}

func screenWindowWndProc(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	switch uMsg {
	case msgCreateWindow:
		p := (*newWindowParams)(unsafe.Pointer(lParam))
		p.w, p.err = newWindow(p.opts)
		return 0
	}
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}

const screenWindowClass = "shiny_ScreenWindow"

func initScreenWindow() (err error) {
	swc, err := syscall.UTF16PtrFromString(screenWindowClass)
	if err != nil {
		return err
	}
	emptyString, err := syscall.UTF16PtrFromString("")
	if err != nil {
		return err
	}
	wc := _WNDCLASS{
		LpszClassName: swc,
		LpfnWndProc:   syscall.NewCallback(screenWindowWndProc),
		HIcon:         hDefaultIcon,
		HCursor:       hDefaultCursor,
		HInstance:     hThisInstance,
		HbrBackground: syscall.Handle(_COLOR_BTNFACE + 1),
	}
	_, err = _RegisterClass(&wc)
	if err != nil {
		return err
	}
	screenHWND, err = _CreateWindowEx(0,
		swc, emptyString,
		_WS_OVERLAPPEDWINDOW,
		_CW_USEDEFAULT, _CW_USEDEFAULT,
		_CW_USEDEFAULT, _CW_USEDEFAULT,
		_HWND_MESSAGE, 0, hThisInstance, 0)
	if err != nil {
		return err
	}
	return nil
}
