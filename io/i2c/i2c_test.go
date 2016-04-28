// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package i2c

import (
	"testing"
)

func TestTenBit(t *testing.T) {
	tc := []struct {
		masked int
		addr   int
		tenbit bool
	}{
		{TenBit(0x5), 0x5, true},
		{0x5, 0x5, false},
		{TenBit(0x40), 0x40, true},
	}

	for _, tt := range tc {
		unmasked, tenbit := resolveAddr(tt.masked)
		if want, got := tt.tenbit, tenbit; got != want {
			t.Errorf("want address %b as 10-bit; got non 10-bit", tt.addr)
		}
		if want, got := tt.addr, unmasked; got != want {
			t.Errorf("want address %b; got %b", want, got)
		}
	}
}
