// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locale

import "testing"

func TestRegionDistance(t *testing.T) {
	tests := []struct {
		a, b string
		d    int
	}{
		{"NL", "NL", 0},
		{"NL", "EU", 1},
		{"EU", "NL", 1},
		{"005", "005", 0},
		{"NL", "BE", 2},
		{"CO", "005", 1},
		{"005", "CO", 1},
		{"CO", "419", 2},
		{"419", "CO", 2},
		{"005", "419", 1},
		{"419", "005", 1},
		{"001", "013", 2},
		{"013", "001", 2},
		{"CO", "CW", 4},
		{"CO", "PW", 6},
		{"CO", "BV", 6},
	}
	for i, tt := range tests {
		if d := regionDistance(getRegionID([]byte(tt.a)), getRegionID([]byte(tt.b))); d != tt.d {
			t.Errorf("%d: d(%s, %s) = %v; want %v", i, tt.a, tt.b, d, tt.d)
		}
	}
}
