// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

#include "windriver.h"

void mainMessagePump(void) {
	MSG msg;

	// This GetMessage cannot fail: http://blogs.msdn.com/b/oldnewthing/archive/2013/03/22/10404367.aspx
	// TODO(andlabs): besides, what should we do if a future Windows change makes it fail for some other reason? we can't return an error because it's too late to stop the main function
	while (GetMessage(&msg, NULL, 0, 0)) {
		TranslateMessage(&msg);
		DispatchMessage(&msg);
	}
}

HRESULT lastErrorToHRESULT(void) {
	DWORD le;

	le = GetLastError();
	if (le == 0)
		return E_FAIL;
	return HRESULT_FROM_WIN32(le);
}
