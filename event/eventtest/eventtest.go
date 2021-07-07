// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package eventtest supports logging events to a test.
// You can use NewContext to create a context that knows how to deliver
// telemetry events back to the test.
// You must use this context or a derived one anywhere you want telemetry to be
// correctly routed back to the test it was constructed with.
package eventtest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
)

// NewContext returns a context you should use for the active test.
func NewContext(ctx context.Context, tb testing.TB) context.Context {
	h := &testHandler{tb: tb}
	return event.WithExporter(ctx, event.NewExporter(h, nil))
}

type testHandler struct {
	tb      testing.TB
	printer logfmt.Printer
}

func (h *testHandler) Event(ctx context.Context, ev *event.Event) context.Context {
	//TODO: choose between stdout and stderr based on the event
	//TODO: decide if we should be calling h.tb.Fail()
	h.printer.Event(os.Stdout, ev)
	return ctx
}

var InitialTime = func() time.Time {
	t, _ := time.Parse(logfmt.TimeFormat, "2020/03/05 14:27:48")
	return t
}()

func ExporterOptions() *event.ExporterOptions {
	nextTime := InitialTime
	return &event.ExporterOptions{
		Now: func() time.Time {
			thisTime := nextTime
			nextTime = nextTime.Add(time.Second)
			return thisTime
		},
	}
}

func CmpOptions() []cmp.Option {
	return []cmp.Option{
		cmpopts.SortSlices(func(x, y event.Label) bool {
			return x.Name < y.Name
		}),
		cmpopts.IgnoreFields(event.Event{}, "At", "ctx", "target", "labels"),
	}
}
