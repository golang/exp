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
		{ErrorLevel, "ERROR"},
		{ErrorLevel + 2, "ERROR+2"},
		{ErrorLevel - 2, "WARN+2"},
		{WarnLevel, "WARN"},
		{WarnLevel - 1, "INFO+3"},
		{InfoLevel, "INFO"},
		{InfoLevel + 1, "INFO+1"},
		{InfoLevel - 3, "DEBUG+1"},
		{DebugLevel, "DEBUG"},
		{DebugLevel - 2, "DEBUG-2"},
	} {
		got := test.in.String()
		if got != test.want {
			t.Errorf("%d: got %s, want %s", test.in, got, test.want)
		}
	}
}

func TestLevelVar(t *testing.T) {
	var al LevelVar
	if got, want := al.Level(), InfoLevel; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	al.Set(WarnLevel)
	if got, want := al.Level(), WarnLevel; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	al.Set(InfoLevel)
	if got, want := al.Level(), InfoLevel; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
