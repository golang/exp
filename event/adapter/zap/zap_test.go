// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package zap_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"
	"golang.org/x/exp/event"
	ezap "golang.org/x/exp/event/adapter/zap"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/keys"
	"golang.org/x/exp/event/severity"
)

func Test(t *testing.T) {
	ctx, h := eventtest.NewCapture()
	log := zap.New(ezap.NewCore(ctx), zap.Fields(zap.Int("traceID", 17), zap.String("resource", "R")))
	log = log.Named("n/m")
	log.Info("mess", zap.Float64("pi", 3.14))
	want := []event.Event{{
		Labels: []event.Label{
			keys.Int64("traceID").Of(17),
			keys.String("resource").Of("R"),
			severity.Info,
			event.Name.Of("n/m"),
			keys.Float64("pi").Of(3.14),
			event.Message.Of("mess"),
		},
	}}
	if diff := cmp.Diff(want, h.Got, cmpopts.IgnoreFields(event.Event{}, "At")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
