// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ezap

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
	"golang.org/x/exp/event/logging/internal"
)

func Test(t *testing.T) {
	e, h := internal.NewTestExporter()
	ctx := event.WithExporter(context.Background(), e)
	log := zap.New(NewCore(ctx), zap.Fields(zap.Int("traceID", 17), zap.String("resource", "R")))
	log = log.Named("n/m")
	log.Info("mess", zap.Float64("pi", 3.14))
	want := &event.Event{
		ID: 1,
		Labels: []event.Label{
			keys.Int64("traceID").Of(17),
			keys.String("resource").Of("R"),
			internal.LevelKey.Of(0),
			internal.NameKey.Of("n/m"),
			keys.Float64("pi").Of(3.14),
			event.Message.Of("mess"),
		},
	}
	h.Got.At = want.At
	if diff := cmp.Diff(want, &h.Got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
