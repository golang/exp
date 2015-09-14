// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

package windriver

import (
	"syscall"
	"unsafe"
)

type _COLORREF uint32

func _RGB(r, g, b byte) _COLORREF {
	return _COLORREF(r) | _COLORREF(g)<<8 | _COLORREF(b)<<16
}

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

type _BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type _RGBQUAD struct {
	Blue     byte
	Green    byte
	Red      byte
	Reserved byte
}

type _BITMAPINFO struct {
	Header _BITMAPINFOHEADER
	Colors [1]_RGBQUAD
}

type _BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

// ToUintptr helps to pass bf to syscall.Syscall.
func (bf _BLENDFUNCTION) ToUintptr() uintptr {
	return *((*uintptr)(unsafe.Pointer(&bf)))
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
	_VK_SHIFT   = 16
	_VK_CONTROL = 17
	_VK_MENU    = 18
	_VK_LWIN    = 0x5B
	_VK_RWIN    = 0x5C
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

const (
	_BI_RGB         = 0
	_DIB_RGB_COLORS = 0

	_AC_SRC_OVER  = 0x00
	_AC_SRC_ALPHA = 0x01
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
//sys	_DeleteDC(dc syscall.Handle) (err error) = user32.DeleteDC
//sys	_CreateDIBSection(dc syscall.Handle, bmi *_BITMAPINFO, usage uint32, bits **byte, section syscall.Handle, offset uint32) (bitmap syscall.Handle, err error) = gdi32.CreateDIBSection
//sys	_CreateCompatibleDC(dc syscall.Handle) (newdc syscall.Handle, err error) = gdi32.CreateCompatibleDC
//sys	_SelectObject(dc syscall.Handle, gdiobj syscall.Handle) (newobj syscall.Handle, err error) = gdi32.SelectObject
//sys	_AlphaBlend(dcdest syscall.Handle, xoriginDest int32, yoriginDest int32, wDest int32, hDest int32, dcsrc syscall.Handle, xoriginSrc int32, yoriginSrc int32, wsrc int32, hsrc int32, ftn uintptr) (err error) = msimg32.AlphaBlend
//sys	_CreateSolidBrush(color _COLORREF) (brush syscall.Handle, err error) = gdi32.CreateSolidBrush
//sys	_FillRect(dc syscall.Handle, rc *_RECT, brush syscall.Handle) (err error) = user32.FillRect
//sys	_DeleteObject(object syscall.Handle) (err error) = gdi32.DeleteObject
//sys	_GetKeyState(virtkey int32) (keystatus int16) = user32.GetKeyState
