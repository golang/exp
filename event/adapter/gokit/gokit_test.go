// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package gokit_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/gokit"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/keys"
)

func Test(t *testing.T) {
	log := gokit.NewLogger()
	ctx, h := eventtest.NewCapture()
	log.Log(ctx, "msg", "mess", "level", 1, "name", "n/m", "traceID", 17, "resource", "R")
	want := []event.Event{{
		At:      eventtest.InitialTime,
		Kind:    event.LogKind,
		Message: "mess",
		Labels: []event.Label{
			keys.Value("level").Of(1),
			keys.Value("name").Of("n/m"),
			keys.Value("traceID").Of(17),
			keys.Value("resource").Of("R"),
		},
	}}
	if diff := cmp.Diff(want, h.Got, eventtest.CmpOption()); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
