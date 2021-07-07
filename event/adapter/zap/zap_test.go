// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package zap_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"golang.org/x/exp/event"
	ezap "golang.org/x/exp/event/adapter/zap"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/severity"
)

func Test(t *testing.T) {
	ctx, h := eventtest.NewCapture()
	log := zap.New(ezap.NewCore(ctx), zap.Fields(zap.Int("traceID", 17), zap.String("resource", "R")))
	log = log.Named("n/m")
	log.Info("mess", zap.Float64("pi", 3.14))
	want := []event.Event{{
		ID:   1,
		Kind: event.LogKind,
		Labels: []event.Label{
			event.Int64("traceID", 17),
			event.String("resource", "R"),
			severity.Info.Label(),
			event.String("name", "n/m"),
			event.Float64("pi", 3.14),
			event.String("msg", "mess"),
		},
	}}
	if diff := cmp.Diff(want, h.Got, eventtest.CmpOptions()...); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
