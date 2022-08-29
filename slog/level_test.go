// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"math"
	"testing"
)

func TestLevelString(t *testing.T) {
	for _, test := range []struct {
		in   Level
		want string
	}{
		{0, "!BADLEVEL(0)"},
		{ErrorLevel, "ERROR"},
		{ErrorLevel - 2, "ERROR-2"},
		{WarnLevel, "WARN"},
		{WarnLevel - 1, "WARN-1"},
		{InfoLevel, "INFO"},
		{InfoLevel - 3, "INFO-3"},
		{DebugLevel, "DEBUG"},
		{InfoLevel + 2, "DEBUG+1"},
		{-1, "!BADLEVEL(-1)"},
	} {
		got := test.in.String()
		if got != test.want {
			t.Errorf("%d: got %s, want %s", test.in, got, test.want)
		}
	}
}

func TestAtomicLevel(t *testing.T) {
	var r *AtomicLevel
	if got, want := r.Level(), Level(math.MaxInt); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	r = NewAtomicLevel(WarnLevel)
	if got, want := r.Level(), WarnLevel; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	r.Set(InfoLevel)
	if got, want := r.Level(), InfoLevel; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
