// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !race

package bench_test

import (
	"testing"

	"golang.org/x/exp/event/bench"
)

func TestLogEventAllocs(t *testing.T) {
	bench.TestAllocs(t, eventPrint, eventLog, 0)
}
