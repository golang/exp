// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package logr_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	elogr "golang.org/x/exp/event/adapter/logr"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/severity"
)

func TestInfo(t *testing.T) {
	ctx, th := eventtest.NewCapture()
	log := elogr.NewLogger(ctx, "/").WithName("n").V(int(severity.Debug))
	log = log.WithName("m")
	log.Info("mess", "traceID", 17, "resource", "R")
	want := []event.Event{{
		ID:   1,
		Kind: event.LogKind,
		Labels: []event.Label{
			severity.Debug.Label(),
			event.Value("traceID", 17),
			event.Value("resource", "R"),
			event.String("name", "n/m"),
			event.String("msg", "mess"),
		},
	}}
	if diff := cmp.Diff(want, th.Got, eventtest.CmpOptions()...); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
