// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "_cgo_export.h"
#include "windriver.h"

static HBITMAP mkbitmap(HDC dc, RECT *r, VOID **ppvBits) {
	BITMAPINFO bi;
	LONG dx, dy;
	HBITMAP bitmap;

	dx = r->right - r->left;
	dy = r->bottom - r->top;

	ZeroMemory(&bi, sizeof (BITMAPINFO));
	bi.bmiHeader.biSize = sizeof (BITMAPINFOHEADER);
	bi.bmiHeader.biWidth = (LONG) dx;
	bi.bmiHeader.biHeight = -((LONG) dy); // negative height to force top-down drawing
	bi.bmiHeader.biPlanes = 1;
	bi.bmiHeader.biBitCount = 32;
	bi.bmiHeader.biCompression = BI_RGB;
	bi.bmiHeader.biSizeImage = (DWORD) (dx * dy * 4);

	bitmap = CreateDIBSection(dc, &bi, DIB_RGB_COLORS, ppvBits, 0, 0);
	if (bitmap == NULL) {
		// TODO(andlabs)
	}
	return bitmap;
}

static void blend(HDC dc, HBITMAP bitmap, RECT *dr, LONG sdx, LONG sdy) {
	HDC compatibleDC;
	HBITMAP prevBitmap;
	BLENDFUNCTION blendfunc;

	compatibleDC = CreateCompatibleDC(dc);
	if (compatibleDC == NULL) {
		// TODO(andlabs)
	}
	prevBitmap = SelectObject(compatibleDC, bitmap);
	if (prevBitmap == NULL) {
		// TODO(andlabs)
	}

	ZeroMemory(&blendfunc, sizeof (BLENDFUNCTION));
	blendfunc.BlendOp = AC_SRC_OVER;
	blendfunc.BlendFlags = 0;
	blendfunc.SourceConstantAlpha = 255;  // only use per-pixel alphas
	blendfunc.AlphaFormat = AC_SRC_ALPHA; // premultiplied
	if (AlphaBlend(dc, dr->left, dr->top,
		dr->right - dr->left, dr->bottom - dr->top,
		compatibleDC, 0, 0, sdx, sdy,
		blendfunc) == FALSE) {
		// TODO
	}

	// TODO(andlabs): error check these?
	SelectObject(compatibleDC, prevBitmap);
	DeleteDC(compatibleDC);
}

// TODO(andlabs): Upload

void fillSrc(PVOID dc, RECT *r, COLORREF color) {
	HBRUSH brush;

	// COLORREF is 0x00BBGGRR; color is 0xAARRGGBB
	color = RGB((color >> 16) & 0xFF,
		(color >> 8) & 0xFF,
		color & 0xFF);
	brush = CreateSolidBrush(color);
	if (brush == NULL) {
		// TODO
	}
	if (FillRect(dc, r, brush) == 0) {
		// TODO
	}
	// TODO(andlabs) check errors?
	DeleteObject(brush);
}

void fillOver(PVOID dc, RECT *r, COLORREF color) {
	HBITMAP bitmap;
	VOID *ppvBits;
	RECT oneByOne;

	// AlphaBlend will stretch the input image (using StretchBlt's
	// COLORONCOLOR mode) to fill the output rectangle. Testing
	// this shows that the result appears to be the same as if we had
	// used a MxN bitmap instead.
	oneByOne.left = 0;
	oneByOne.top = 0;
	oneByOne.right = 1;
	oneByOne.bottom = 1;
	bitmap = mkbitmap(dc, &oneByOne, &ppvBits);
	*((uint32_t *) ppvBits) = color;
	blend(dc, bitmap, r, 1, 1);
	// TODO(andlabs): check errors?
	DeleteObject(bitmap);
}

// TODO(andlabs): Draw
