// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "windriver.h"

HWND utilityWindow = NULL;

static LRESULT CALLBACK utilityWindowWndProc(HWND hwnd, UINT uMsg, WPARAM wParam, LPARAM lParam) {
	HWND *phwnd;

	switch (uMsg) {
	case msgCreateWindow:
		phwnd = (HWND *) lParam;
		return utilCreateWindow(phwnd);
	case msgDestroyWindow:
		return utilDestroyWindow((HWND) wParam);
	}
	return DefWindowProcW(hwnd, uMsg, wParam, lParam);
}

HRESULT initUtilityWindow(void) {
	WNDCLASSW wc;

	ZeroMemory(&wc, sizeof (WNDCLASSW));
	wc.lpszClassName = L"shiny_utilityWindow";
	wc.lpfnWndProc = utilityWindowWndProc;
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

	utilityWindow = CreateWindowExW(0,
		L"shiny_utilityWindow", L"Shiny Utility Window",
		WS_OVERLAPPEDWINDOW,
		CW_USEDEFAULT, CW_USEDEFAULT,
		CW_USEDEFAULT, CW_USEDEFAULT,
		HWND_MESSAGE, NULL, thishInstance, NULL);
	if (utilityWindow == NULL) {
		return lastErrorToHRESULT();
	}

	return S_OK;
}
