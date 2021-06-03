// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package elogrus_test

import (
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/eventtest"
	"golang.org/x/exp/event/keys"
	"golang.org/x/exp/event/logging/elogrus"
	"golang.org/x/exp/event/logging/internal"
)

func Test(t *testing.T) {
	ctx, th := eventtest.NewCapture()
	log := logrus.New()
	log.SetFormatter(elogrus.NewFormatter())
	log.SetOutput(io.Discard)
	// adding WithContext panics, because event.FromContext assumes
	log.WithContext(ctx).WithField("traceID", 17).WithField("resource", "R").Info("mess")

	want := []event.Event{{
		Labels: []event.Label{
			internal.LevelKey.Of(4),
			keys.Value("traceID").Of(17),
			keys.Value("resource").Of("R"),
			event.Message.Of("mess"),
		},
	}}
	// logrus fields are stored in a map, so we have to sort to overcome map
	// iteration indeterminacy.
	less := func(a, b event.Label) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want, th.Got, cmpopts.SortSlices(less), cmpopts.IgnoreFields(event.Event{}, "At")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
