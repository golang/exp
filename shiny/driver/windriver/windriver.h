// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

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

// see http://blogs.msdn.com/b/oldnewthing/archive/2004/10/25/247180.aspx
// this will work on MinGW too
EXTERN_C IMAGE_DOS_HEADER __ImageBase;
#define thishInstance ((HINSTANCE) (&__ImageBase))

// messages sent to the utility window to do the various functions of the package on the UI thread
// we start at WM_USER + 0x40 to make room for the DM_* messages
enum {
	// wParam - 0
	// lParam - pointer to store HWND in
	// return - error LRESULT
	msgCreateWindow = WM_USER + 0x40,
	// wParam - hwnd
	// lParam - 0
	// return - error LRESULT
	msgDestroyWindow,
};

// windriver.c
extern void mainMessagePump(void);
extern HRESULT lastErrorToHRESULT(void);
#define lS_OK ((LRESULT) S_OK)
#define lastErrorToLRESULT() ((LRESULT) lastErrorToHRESULT())

// utilwindow.c
extern HWND utilityWindow;
extern HRESULT initUtilityWindow(void);

// window.c
extern HRESULT initWindowClass(void);
extern HRESULT createWindow(HWND *);
extern LRESULT utilCreateWindow(HWND *);
extern HRESULT destroyWindow(HWND);
extern LRESULT utilDestroyWindow(HWND);
