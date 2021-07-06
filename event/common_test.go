// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
)

func TestCommon(t *testing.T) {
	m := event.NewCounter("m", "")
	for _, test := range []struct {
		method string
		events func(context.Context)
		expect []event.Event
	}{{
		method: "Log",
		events: func(ctx context.Context) { event.Log(ctx, "simple message") },
		expect: []event.Event{{
			ID:     1,
			Kind:   event.LogKind,
			Labels: []event.Label{event.String("msg", "simple message")},
		}},
	}, {
		method: "Logf",
		events: func(ctx context.Context) { event.Logf(ctx, "logf %s message", "to") },
		expect: []event.Event{{
			ID:     1,
			Kind:   event.LogKind,
			Labels: []event.Label{event.String("msg", "logf to message")},
		}},
	}, {
		method: "Metric",
		events: func(ctx context.Context) {
			m.Record(ctx, 3)
		},
		expect: []event.Event{{
			ID:   1,
			Kind: event.MetricKind,
			Labels: []event.Label{
				event.Int64("metricValue", 3),
				event.Value("metric", m),
			},
		}},
	}, {
		method: "Annotate",
		events: func(ctx context.Context) { event.Annotate(ctx, event.String("other", "some value")) },
		expect: []event.Event{{
			ID:     1,
			Labels: []event.Label{event.String("other", "some value")},
		}},
	}, {
		method: "Start",
		events: func(ctx context.Context) {
			ctx = event.Start(ctx, `a trace`)
			event.End(ctx)
		},
		expect: []event.Event{{
			ID:     1,
			Kind:   event.StartKind,
			Labels: []event.Label{event.String("name", "a trace")},
		}, {
			ID:     2,
			Parent: 1,
			Kind:   event.EndKind,
			Labels: []event.Label{},
		}},
	}} {
		t.Run(test.method, func(t *testing.T) {
			ctx, h := eventtest.NewCapture()
			test.events(ctx)
			if diff := cmp.Diff(test.expect, h.Got, eventtest.CmpOption()); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
