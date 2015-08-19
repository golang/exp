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

	switch (uMsg) {
	case WM_PAINT:
		// TODO
		break;
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
	return (HRESULT) SendMessageW(utilityWindow, msgCreateWindow, 0, (LPARAM) phwnd);
}

LRESULT utilCreateWindow(HWND *phwnd) {
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
	// TODO(andlabs): UpdateWindow()?
	return lS_OK;
}

HRESULT destroyWindow(HWND hwnd) {
	return (HRESULT) SendMessageW(utilityWindow, msgDestroyWindow, (WPARAM) hwnd, 0);
}

LRESULT utilDestroyWindow(HWND hwnd) {
	if (DestroyWindow(hwnd) == 0) {
		return lastErrorToLRESULT();
	}
	return lS_OK;
}
