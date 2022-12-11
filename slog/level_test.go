// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"testing"
)

func TestLevelString(t *testing.T) {
	for _, test := range []struct {
		in   Level
		want string
	}{
		{0, "INFO"},
		{LevelError, "ERROR"},
		{LevelError + 2, "ERROR+2"},
		{LevelError - 2, "WARN+2"},
		{LevelWarn, "WARN"},
		{LevelWarn - 1, "INFO+3"},
		{LevelInfo, "INFO"},
		{LevelInfo + 1, "INFO+1"},
		{LevelInfo - 3, "DEBUG+1"},
		{LevelDebug, "DEBUG"},
		{LevelDebug - 2, "DEBUG-2"},
	} {
		got := test.in.String()
		if got != test.want {
			t.Errorf("%d: got %s, want %s", test.in, got, test.want)
		}
	}
}

func TestLevelVar(t *testing.T) {
	var al LevelVar
	if got, want := al.Level(), LevelInfo; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	al.Set(LevelWarn)
	if got, want := al.Level(), LevelWarn; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	al.Set(LevelInfo)
	if got, want := al.Level(), LevelInfo; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
