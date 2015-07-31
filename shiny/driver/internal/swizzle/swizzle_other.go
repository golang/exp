// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !amd64

package swizzle

const (
	haveSIMD16 = false
)

func bgra16(p []byte) { panic("unreachable") }
