// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Use the nopc flag for benchmarks, on the assumption
// that retrieving the pc will become cheap.

//go:build nopc

package slog

// pc returns 0 to avoid incurring the cost of runtime.Callers.
func pc(depth int) uintptr { return 0 }
