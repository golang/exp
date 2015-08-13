// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package windriver

// #include "windriver.h"
import "C"

import (
	"fmt"
)

func winerror(msg string, hr C.HRESULT) error {
	// TODO(andlabs): get long description
	if hr == C.E_FAIL {
		return fmt.Errorf("windriver: %s: unknown error", msg)
	}
	return fmt.Errorf("windriver: %s: last error %d", msg, hr&0xFFFF)
}
