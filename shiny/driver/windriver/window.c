// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

#include "_cgo_export.h"
#include "windriver.h"

#define windowClass L"shiny_Window"

static LRESULT CALLBACK windowWndProc(HWND hwnd, UINT uMsg, WPARAM wParam, LPARAM lParam) {
	WINDOWPOS *wp = (WINDOWPOS *) lParam;
	RECT r;
	HDC dc;

	switch (uMsg) {
	case WM_PAINT:
		handlePaint(hwnd);
		break; // explicitly defer to DefWindowProc; it will handle validation for us
	case WM_WINDOWPOSCHANGED:
		if ((wp->flags & SWP_NOSIZE) != 0) {
			break;
		}
		if (GetClientRect(hwnd, &r) == 0) {
			/* TODO(andlabs) */;
		}
		sendSizeEvent(hwnd, &r);
		return 0;
	case WM_MOUSEMOVE:
	case WM_LBUTTONDOWN:
		// TODO(andlabs): call SetFocus()?
	case WM_LBUTTONUP:
	case WM_MBUTTONDOWN:
	case WM_MBUTTONUP:
	case WM_RBUTTONDOWN:
	case WM_RBUTTONUP:
		sendMouseEvent(hwnd, uMsg,
			GET_X_LPARAM(lParam),
			GET_Y_LPARAM(lParam));
		return 0;
	case WM_KEYDOWN:
	case WM_KEYUP:
	case WM_SYSKEYDOWN:
	case WM_SYSKEYUP:
		// TODO
		break;
	case msgFillSrc:
		// TODO error checks
		dc = GetDC(hwnd);
		fillSrc(dc, (RECT *) lParam, (COLORREF) wParam);
		ReleaseDC(hwnd, dc);
		break;
	case msgFillOver:
		// TODO error checks
		dc = GetDC(hwnd);
		fillOver(dc, (RECT *) lParam, (COLORREF) wParam);
		ReleaseDC(hwnd, dc);
		break;
	}
	return DefWindowProcW(hwnd, uMsg, wParam, lParam);
}

HRESULT initWindowClass(void) {
	WNDCLASSW wc;

	ZeroMemory(&wc, sizeof (WNDCLASSW));
	wc.lpszClassName = windowClass;
	wc.lpfnWndProc = windowWndProc;
	wc.hInstance = thishInstance;
	wc.hIcon = LoadIconW(NULL, IDI_APPLICATION);
	if (wc.hIcon == NULL) {
		return lastErrorToHRESULT();
	}
	wc.hCursor = LoadCursorW(NULL, IDC_ARROW);
	if (wc.hCursor == NULL) {
		return lastErrorToHRESULT();
	}
	// TODO(andlabs): change this to something else? NULL? the hollow brush?
	wc.hbrBackground = (HBRUSH) (COLOR_BTNFACE + 1);
	if (RegisterClassW(&wc) == 0) {
		return lastErrorToHRESULT();
	}
	return S_OK;
}

HRESULT createWindow(HWND *phwnd) {
	*phwnd = CreateWindowExW(0,
		windowClass, L"Shiny Window",
		WS_OVERLAPPEDWINDOW,
		CW_USEDEFAULT, CW_USEDEFAULT,
		CW_USEDEFAULT, CW_USEDEFAULT,
		NULL, NULL, thishInstance, NULL);
	if (*phwnd == NULL) {
		return lastErrorToLRESULT();
	}
	// TODO(andlabs): use proper nCmdShow
	ShowWindow(*phwnd, SW_SHOWDEFAULT);
	// TODO(andlabs): call UpdateWindow()
	return lS_OK;
}

void sendFill(HWND hwnd, UINT uMsg, RECT r, COLORREF color) {
	// Note: this SendMessageW won't return until after the fill
	// completes, so using &r is safe.
	SendMessageW(hwnd, uMsg, (WPARAM) color, (LPARAM) (&r));
}
