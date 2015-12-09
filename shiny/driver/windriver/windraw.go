// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

import (
	"syscall"
	"unsafe"

	"golang.org/x/exp/shiny/driver/internal/win32"
)

func mkbitmap(dx, dy int32) (syscall.Handle, *byte, error) {
	bi := _BITMAPINFO{
		Header: _BITMAPINFOHEADER{
			Size:        uint32(unsafe.Sizeof(_BITMAPINFOHEADER{})),
			Width:       dx,
			Height:      -dy, // negative height to force top-down drawing
			Planes:      1,
			BitCount:    32,
			Compression: _BI_RGB,
			SizeImage:   uint32(dx * dy * 4),
		},
	}

	var ppvBits *byte
	bitmap, err := _CreateDIBSection(0, &bi, _DIB_RGB_COLORS, &ppvBits, 0, 0)
	if err != nil {
		return 0, nil, err
	}
	return bitmap, ppvBits, nil
}

func blend(dc win32.HDC, bitmap syscall.Handle, dr *_RECT, sdx int32, sdy int32) (err error) {
	compatibleDC, err := _CreateCompatibleDC(dc)
	if err != nil {
		return err
	}
	defer func() {
		err2 := _DeleteDC(compatibleDC)
		if err == nil {
			err = err2
		}
	}()
	prevBitmap, err := _SelectObject(compatibleDC, bitmap)
	if err != nil {
		return err
	}

	blendfunc := _BLENDFUNCTION{
		BlendOp:             _AC_SRC_OVER,
		BlendFlags:          0,
		SourceConstantAlpha: 255,           // only use per-pixel alphas
		AlphaFormat:         _AC_SRC_ALPHA, // premultiplied
	}
	err = _AlphaBlend(dc, dr.Left, dr.Top,
		dr.Right-dr.Left, dr.Bottom-dr.Top,
		compatibleDC, 0, 0, sdx, sdy,
		blendfunc.ToUintptr())
	if err != nil {
		return err
	}

	_, err = _SelectObject(compatibleDC, prevBitmap)
	return err
}

func blit(dc win32.HDC, dp _POINT, bitmap syscall.Handle, sr *_RECT) (err error) {
	compatibleDC, err := _CreateCompatibleDC(dc)
	if err != nil {
		return err
	}
	defer func() {
		err2 := _DeleteDC(compatibleDC)
		if err == nil {
			err = err2
		}
	}()
	prevBitmap, err := _SelectObject(compatibleDC, bitmap)
	if err != nil {
		return err
	}

	dx, dy := sr.Right-sr.Left, sr.Bottom-sr.Top
	_, err = _BitBlt(dc, dp.X, dp.Y, dx, dy, compatibleDC, sr.Left, sr.Top, _SRCCOPY)
	if err != nil {
		return err
	}
	_, err = _SelectObject(compatibleDC, prevBitmap)
	return err
}

func fillSrc(hwnd win32.HWND, uMsg uint32, wParam, lParam uintptr) {
	dc, err := win32.GetDC(hwnd)
	if err != nil {
		panic(err) // TODO handle error?
	}
	defer win32.ReleaseDC(hwnd, dc)
	r := (*_RECT)(unsafe.Pointer(lParam))
	color := _COLORREF(wParam)

	// COLORREF is 0x00BBGGRR; color is 0xAARRGGBB
	color = _RGB(byte((color >> 16)), byte((color >> 8)), byte(color))
	brush, err := _CreateSolidBrush(color)
	if err != nil {
		panic(err) // TODO handle error
	}
	defer _DeleteObject(brush)
	err = _FillRect(dc, r, brush)
	if err != nil {
		panic(err) // TODO handle error
	}
}

func fillOver(hwnd win32.HWND, uMsg uint32, wParam, lParam uintptr) {
	dc, err := win32.GetDC(hwnd)
	if err != nil {
		panic(err) // TODO handle error
	}
	defer win32.ReleaseDC(hwnd, dc)
	r := (*_RECT)(unsafe.Pointer(lParam))
	color := _COLORREF(wParam)

	// AlphaBlend will stretch the input image (using StretchBlt's
	// COLORONCOLOR mode) to fill the output rectangle. Testing
	// this shows that the result appears to be the same as if we had
	// used a MxN bitmap instead.
	bitmap, bitvalues, err := mkbitmap(1, 1)
	if err != nil {
		panic(err) // TODO handle error
	}
	defer _DeleteObject(bitmap) // TODO handle error?
	*(*_COLORREF)(unsafe.Pointer(bitvalues)) = color
	if err = blend(dc, bitmap, r, 1, 1); err != nil {
		panic(err) // TODO handle error
	}
}

var (
	msgFillSrc  = win32.AddWindowMsg(fillSrc)
	msgFillOver = win32.AddWindowMsg(fillOver)
	msgUpload   = win32.AddWindowMsg(handleUpload)
)

// TODO(andlabs): Draw
