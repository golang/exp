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
)

func Test(t *testing.T) {
	log := gokit.NewLogger()
	ctx, h := eventtest.NewCapture()
	log.Log(ctx, "msg", "mess", "level", 1, "name", "n/m", "traceID", 17, "resource", "R")
	want := []event.Event{{
		ID:   1,
		Kind: event.LogKind,
		Labels: []event.Label{
			event.Value("level", 1),
			event.Value("name", "n/m"),
			event.Value("traceID", 17),
			event.Value("resource", "R"),
			event.String("msg", "mess"),
		},
	}}
	if diff := cmp.Diff(want, h.Got, eventtest.CmpOptions()...); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
