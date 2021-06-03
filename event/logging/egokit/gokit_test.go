// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package egokit_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
	"golang.org/x/exp/event/logging/egokit"
	"golang.org/x/exp/event/logging/internal"
)

func Test(t *testing.T) {
	log := egokit.NewLogger()
	e, h := internal.NewTestExporter()
	ctx := event.WithExporter(context.Background(), e)
	log.Log(ctx, "msg", "mess", "level", 1, "name", "n/m", "traceID", 17, "resource", "R")
	want := &event.Event{
		At: internal.TestAt,
		Labels: []event.Label{
			keys.Value("level").Of(1),
			keys.Value("name").Of("n/m"),
			keys.Value("traceID").Of(17),
			keys.Value("resource").Of("R"),
			event.Message.Of("mess"),
		},
	}
	if diff := cmp.Diff(want, &h.Got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
