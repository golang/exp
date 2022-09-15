// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package buffer

import "testing"

func Test(t *testing.T) {
	b := New()
	defer b.Free()
	b.WriteString("hello")
	b.WriteByte(',')
	b.Write([]byte(" world"))
	got := string(*b)
	want := "hello, world"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
