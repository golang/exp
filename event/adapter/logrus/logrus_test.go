// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package logrus_test

import (
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/event"
	elogrus "golang.org/x/exp/event/adapter/logrus"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/severity"
)

func Test(t *testing.T) {
	ctx, th := eventtest.NewCapture()
	log := logrus.New()
	log.SetFormatter(elogrus.NewFormatter())
	log.SetOutput(io.Discard)
	// adding WithContext panics, because event.FromContext assumes
	log.WithContext(ctx).WithField("traceID", 17).WithField("resource", "R").Info("mess")

	want := []event.Event{{
		ID:   1,
		Kind: event.LogKind,
		Labels: []event.Label{
			severity.Info.Label(),
			event.Value("traceID", 17),
			event.Value("resource", "R"),
			event.String("msg", "mess"),
		},
	}}
	// logrus fields are stored in a map, so we have to sort to overcome map
	// iteration indeterminacy.
	if diff := cmp.Diff(want, th.Got, eventtest.CmpOptions()...); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
