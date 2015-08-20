// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

#include "windriver.h"

HRESULT lastErrorToHRESULT(void) {
	DWORD le;

	le = GetLastError();
	if (le == 0)
		return E_FAIL;
	return HRESULT_FROM_WIN32(le);
}
