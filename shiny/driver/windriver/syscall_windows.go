// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

package windriver

import "syscall"

type _COLORREF uint32

type _POINT struct {
	X int32
	Y int32
}

type _RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type _MSG struct {
	Hwnd    syscall.Handle
	Message uint32
	Wparam  uintptr
	Lparam  uintptr
	Time    uint32
	Pt      _POINT
}

type _WNDCLASS struct {
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
}

type _WINDOWPOS struct {
	Hwnd            syscall.Handle
	HwndInsertAfter syscall.Handle
	X               int32
	Y               int32
	Cx              int32
	Cy              int32
	Flags           uint32
}

const (
	_WM_PAINT            = 15
	_WM_WINDOWPOSCHANGED = 71
	_WM_KEYDOWN          = 256
	_WM_KEYUP            = 257
	_WM_SYSKEYDOWN       = 260
	_WM_SYSKEYUP         = 261
	_WM_MOUSEMOVE        = 512
	_WM_LBUTTONDOWN      = 513
	_WM_LBUTTONUP        = 514
	_WM_RBUTTONDOWN      = 516
	_WM_RBUTTONUP        = 517
	_WM_MBUTTONDOWN      = 519
	_WM_MBUTTONUP        = 520
	_WM_USER             = 0x0400
)

const (
	_WS_OVERLAPPED       = 0x00000000
	_WS_CAPTION          = 0x00C00000
	_WS_SYSMENU          = 0x00080000
	_WS_THICKFRAME       = 0x00040000
	_WS_MINIMIZEBOX      = 0x00020000
	_WS_MAXIMIZEBOX      = 0x00010000
	_WS_OVERLAPPEDWINDOW = _WS_OVERLAPPED | _WS_CAPTION | _WS_SYSMENU | _WS_THICKFRAME | _WS_MINIMIZEBOX | _WS_MAXIMIZEBOX
)

const (
	_COLOR_BTNFACE = 15
)

const (
	_IDI_APPLICATION = 32512
	_IDC_ARROW       = 32512
)

const (
	_CW_USEDEFAULT = 0x80000000 - 0x100000000

	_SW_SHOWDEFAULT = 10

	_HWND_MESSAGE = syscall.Handle(^uintptr(2)) // -3

	_SWP_NOSIZE = 0x0001
)

func _GET_X_LPARAM(lp uintptr) int32 {
	return int32(_LOWORD(lp))
}

func _GET_Y_LPARAM(lp uintptr) int32 {
	return int32(_HIWORD(lp))
}

func _LOWORD(l uintptr) uint16 {
	return uint16(uint32(l))
}

func _HIWORD(l uintptr) uint16 {
	return uint16(uint32(l >> 16))
}

// notes to self
// UINT = uint32
// callbacks = uintptr
// strings = *uint16

//sys	_GetMessage(msg *_MSG, hwnd syscall.Handle, msgfiltermin uint32, msgfiltermax uint32) (ret int32, err error) [failretval==-1] = user32.GetMessageW
//sys	_TranslateMessage(msg *_MSG) (done bool) = user32.TranslateMessage
//sys	_DispatchMessage(msg *_MSG) (ret int32) = user32.DispatchMessageW
//sys	_DefWindowProc(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) = user32.DefWindowProcW
//sys	_RegisterClass(wc *_WNDCLASS) (atom uint16, err error) = user32.RegisterClassW
//sys	_CreateWindowEx(exstyle uint32, className *uint16, windowText *uint16, style uint32, x int32, y int32, width int32, height int32, parent syscall.Handle, menu syscall.Handle, hInstance syscall.Handle, lpParam uintptr) (hwnd syscall.Handle, err error) = user32.CreateWindowExW
//sys	_DestroyWindow(hwnd syscall.Handle) (err error) = user32.DestroyWindow
//sys	_SendMessage(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) = user32.SendMessageW
//sys	_LoadIcon(hInstance syscall.Handle, iconName uintptr) (icon syscall.Handle, err error) = user32.LoadIconW
//sys	_LoadCursor(hInstance syscall.Handle, cursorName uintptr) (cursor syscall.Handle, err error) = user32.LoadCursorW
//sys	_ShowWindow(hwnd syscall.Handle, cmdshow int32) (wasvisible bool) = user32.ShowWindow
//sys	_GetClientRect(hwnd syscall.Handle, rect *_RECT) (err error) = user32.GetClientRect
//sys	_GetDC(hwnd syscall.Handle) (dc syscall.Handle, err error) = user32.GetDC
//sys	_ReleaseDC(hwnd syscall.Handle, dc syscall.Handle) (err error) = user32.ReleaseDC
