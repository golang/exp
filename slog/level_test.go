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
		{LevelQuiet, "QUIET"}, // lowest int8 -128
		{LevelQuiet + 1, "ERROR-41"},
		{LevelQuiet + 42, "ERROR"},
		{LevelError - 41, "ERROR-41"},

		{LevelError, "ERROR"},
		{LevelError - 42, "QUIET"},
		{LevelError - 41, "ERROR-41"},
		{LevelError - 1, "ERROR-1"},
		{LevelError + 1, "ERROR+1"},
		{LevelError + 41, "ERROR+41"},
		{LevelError + 42, "WARN"},

		{LevelWarn, "WARN"},
		{LevelWarn - 42, "ERROR"},
		{LevelWarn - 41, "ERROR+1"},
		{LevelWarn - 1, "ERROR+41"},
		{LevelWarn + 1, "WARN+1"},
		{LevelWarn + 42, "WARN+42"},
		{LevelWarn + 44, "NOTICE"},

		{0, "NOTICE"},
		{LevelNotice, "NOTICE"},
		{LevelNotice - 1, "WARN+43"},
		{LevelNotice + 1, "NOTICE+1"},
		{LevelNotice + 42, "INFO"},

		{LevelInfo, "INFO"},
		{LevelInfo - 1, "NOTICE+41"},
		{LevelInfo + 1, "INFO+1"},

		{LevelDebug, "DEBUG"},
		{LevelDebug - 1, "INFO+41"},
		{LevelDebug + 1, "DEBUG+1"},
	} {
		got := test.in.String()
		if got != test.want {
			t.Errorf("%d: got %s, want %s", test.in, got, test.want)
		}
	}
}

func TestLevelVar(t *testing.T) {
	var al LevelVar
	if got, want := al.Level(), LevelNotice; got != want {
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
