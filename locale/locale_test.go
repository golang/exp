// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locale

import (
	"reflect"
	"testing"
)

func TestIDSize(t *testing.T) {
	id := ID{}
	typ := reflect.TypeOf(id)
	if typ.Size() > 16 {
		t.Errorf("size of ID was %d; want 16", typ.Size())
	}
}

func TestIsRoot(t *testing.T) {
	for i, tt := range parseTests() {
		loc, _ := Parse(tt.in)
		undef := tt.lang == "und" && tt.script == "" && tt.region == "" && tt.ext == ""
		if loc.IsRoot() != undef {
			t.Errorf("%d: was %v; want %v", i, loc.IsRoot(), undef)
		}
	}
}
