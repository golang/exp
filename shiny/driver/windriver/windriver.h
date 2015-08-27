// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

#ifndef __SHINY_WINDRIVER_H__
#define __SHINY_WINDRIVER_H__

#define UNICODE
#define _UNICODE
#define STRICT
#define STRICT_TYPED_ITEMIDS
#define CINTERFACE
#define COBJMACROS
// see https://github.com/golang/go/issues/9916#issuecomment-74812211
#define INITGUID
// get Windows version right; right now Windows XP
#define WINVER 0x0501
#define _WIN32_WINNT 0x0501
#define _WIN32_WINDOWS 0x0501		/* according to Microsoft's winperf.h */
#define _WIN32_IE 0x0600			/* according to Microsoft's sdkddkver.h */
#define NTDDI_VERSION 0x05010000	/* according to Microsoft's sdkddkver.h */
#include <windows.h>
#include <windowsx.h>
#include <stdint.h>

// see http://blogs.msdn.com/b/oldnewthing/archive/2004/10/25/247180.aspx
// this will work on MinGW too
EXTERN_C IMAGE_DOS_HEADER __ImageBase;
#define thishInstance ((HINSTANCE) (&__ImageBase))

#define firstClassMessage (WM_USER + 0x40)

// screen.Window private messages.
// TODO elaborate
enum {
	// for both of these:
	// wParam - COLORREF
	// lParam - pointer to RECT
	msgFillSrc = WM_USER + 0x20,
	msgFillOver,
};

// windriver.c
extern HRESULT lastErrorToHRESULT(void);
#define lS_OK ((LRESULT) S_OK)
#define lastErrorToLRESULT() ((LRESULT) lastErrorToHRESULT())

// window.c
extern HRESULT initWindowClass(void);
extern HRESULT createWindow(HWND *);
extern void sendFill(HWND, UINT, RECT, COLORREF);

// windraw.c
extern void fillSrc(HDC, RECT *, COLORREF);
extern void fillOver(HDC, RECT *, COLORREF);

#endif
