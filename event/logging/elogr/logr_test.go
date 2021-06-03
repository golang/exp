// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package elogr_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
	"golang.org/x/exp/event/logging/elogr"
	"golang.org/x/exp/event/logging/internal"
)

func TestInfo(t *testing.T) {
	e, th := internal.NewTestExporter()
	log := elogr.NewLogger(event.WithExporter(context.Background(), e), "/").WithName("n").V(3)
	log = log.WithName("m")
	log.Info("mess", "traceID", 17, "resource", "R")
	want := &event.Event{
		At: internal.TestAt,
		Labels: []event.Label{
			internal.LevelKey.Of(3),
			internal.NameKey.Of("n/m"),
			keys.Value("traceID").Of(17),
			keys.Value("resource").Of("R"),
			event.Message.Of("mess"),
		},
	}
	if diff := cmp.Diff(want, &th.Got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
