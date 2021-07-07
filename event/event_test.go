// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/eventtest"
)

var (
	l1      = event.Int64("l1", 1)
	l2      = event.Int64("l2", 2)
	l3      = event.Int64("l3", 3)
	counter = event.NewCounter("hits", "cache hits")
	gauge   = event.NewFloatGauge("temperature", "CPU board temperature in Celsius")
	latency = event.NewDuration("latency", "how long it took")
	err     = errors.New("an error")
)

func TestCommon(t *testing.T) {
	for _, test := range []struct {
		method string
		events func(context.Context)
		expect []event.Event
	}{{
		method: "simple",
		events: func(ctx context.Context) { event.Log(ctx, "a message") },
		expect: []event.Event{{
			ID:     1,
			Kind:   event.LogKind,
			Labels: []event.Label{event.String("msg", "a message")},
		}},
	}, {
		method: "log 1",
		events: func(ctx context.Context) { event.Log(ctx, "a message", l1) },
		expect: []event.Event{{
			ID:     1,
			Kind:   event.LogKind,
			Labels: []event.Label{event.String("msg", "a message"), l1},
		}},
	}, {
		method: "log 2",
		events: func(ctx context.Context) { event.Log(ctx, "a message", l1, l2) },
		expect: []event.Event{{
			ID:     1,
			Kind:   event.LogKind,
			Labels: []event.Label{event.String("msg", "a message"), l1, l2},
		}},
	}, {
		method: "log 3",
		events: func(ctx context.Context) { event.Log(ctx, "a message", l1, l2, l3) },
		expect: []event.Event{{
			ID:     1,
			Kind:   event.LogKind,
			Labels: []event.Label{event.String("msg", "a message"), l1, l2, l3},
		}},
	}, {
		method: "logf",
		events: func(ctx context.Context) { event.Logf(ctx, "logf %s message", "to") },
		expect: []event.Event{{
			ID:     1,
			Kind:   event.LogKind,
			Labels: []event.Label{event.String("msg", "logf to message")},
		}},
	}, {
		method: "error",
		events: func(ctx context.Context) { event.Error(ctx, "failed", err, l1) },
		expect: []event.Event{{
			ID:   1,
			Kind: event.LogKind,
			Labels: []event.Label{
				event.String("msg", "failed"),
				event.Value("error", err),
				l1,
			},
		}},
	}, {
		method: "span",
		events: func(ctx context.Context) {
			ctx = event.Start(ctx, `span`)
			event.End(ctx)
		},
		expect: []event.Event{{
			ID:     1,
			Kind:   event.StartKind,
			Labels: []event.Label{event.String("name", "span")},
		}, {
			ID:     2,
			Parent: 1,
			Kind:   event.EndKind,
			Labels: []event.Label{},
		}},
	}, {
		method: "span nested",
		events: func(ctx context.Context) {
			ctx = event.Start(ctx, "parent")
			defer event.End(ctx)
			child := event.Start(ctx, "child")
			defer event.End(child)
			event.Log(child, "message")
		},
		expect: []event.Event{{
			ID:     1,
			Kind:   event.StartKind,
			Labels: []event.Label{event.String("name", "parent")},
		}, {
			ID:     2,
			Parent: 1,
			Kind:   event.StartKind,
			Labels: []event.Label{event.String("name", "child")},
		}, {
			ID:     3,
			Parent: 2,
			Kind:   event.LogKind,
			Labels: []event.Label{event.String("msg", "message")},
		}, {
			ID:     4,
			Parent: 2,
			Kind:   event.EndKind,
			Labels: []event.Label{},
		}, {
			ID:     5,
			Parent: 1,
			Kind:   event.EndKind,
			Labels: []event.Label{},
		}},
	}, {
		method: "counter",
		events: func(ctx context.Context) { counter.Record(ctx, 2, l1) },
		expect: []event.Event{{
			ID:   1,
			Kind: event.MetricKind,
			Labels: []event.Label{
				event.Int64("metricValue", 2),
				event.Value("metric", counter),
				l1,
			},
		}},
	}, {
		method: "gauge",
		events: func(ctx context.Context) { gauge.Record(ctx, 98.6, l1) },
		expect: []event.Event{{
			ID:   1,
			Kind: event.MetricKind,
			Labels: []event.Label{
				event.Float64("metricValue", 98.6),
				event.Value("metric", gauge),
				l1,
			},
		}},
	}, {
		method: "duration",
		events: func(ctx context.Context) { latency.Record(ctx, 3*time.Second, l1, l2) },
		expect: []event.Event{{
			ID:   1,
			Kind: event.MetricKind,
			Labels: []event.Label{
				event.Duration("metricValue", 3*time.Second),
				event.Value("metric", latency),
				l1, l2,
			},
		}},
	}, {
		method: "annotate",
		events: func(ctx context.Context) { event.Annotate(ctx, l1) },
		expect: []event.Event{{
			ID:     1,
			Labels: []event.Label{l1},
		}},
	}, {
		method: "annotate 2",
		events: func(ctx context.Context) { event.Annotate(ctx, l1, l2) },
		expect: []event.Event{{
			ID:     1,
			Labels: []event.Label{l1, l2},
		}},
	}, {
		method: "multiple events",
		events: func(ctx context.Context) {
			/*TODO: this is supposed to be using a cached target
			t := event.To(ctx)
			p := event.Prototype{}.As(event.LogKind)
			t.With(p).Int("myInt", 6).Message("my event").Send()
			t.With(p).String("myString", "some string value").Message("string event").Send()
			*/
			event.Log(ctx, "my event", event.Int64("myInt", 6))
			event.Log(ctx, "string event", event.String("myString", "some string value"))
		},
		expect: []event.Event{{
			ID:   1,
			Kind: event.LogKind,
			Labels: []event.Label{
				event.String("msg", "my event"),
				event.Int64("myInt", 6),
			},
		}, {
			ID:   2,
			Kind: event.LogKind,
			Labels: []event.Label{
				event.String("msg", "string event"),
				event.String("myString", "some string value"),
			},
		}},
	}} {
		t.Run(test.method, func(t *testing.T) {
			ctx, h := eventtest.NewCapture()
			test.events(ctx)
			if diff := cmp.Diff(test.expect, h.Got, eventtest.CmpOptions()...); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func ExampleLog() {
	ctx := event.WithExporter(context.Background(), event.NewExporter(logfmt.NewHandler(os.Stdout), eventtest.ExporterOptions()))
	event.Log(ctx, "my event", event.Int64("myInt", 6))
	event.Log(ctx, "error event", event.String("myString", "some string value"))
	// Output:
	// time="2020/03/05 14:27:48" myInt=6 msg="my event"
	// time="2020/03/05 14:27:49" myString="some string value" msg="error event"
}

func TestLogEventf(t *testing.T) {
	eventtest.TestBenchmark(t, eventPrint, eventLogf, eventtest.LogfOutput)
}

func TestLogEvent(t *testing.T) {
	eventtest.TestBenchmark(t, eventPrint, eventLog, eventtest.LogfmtOutput)
}

func TestTraceBuilder(t *testing.T) {
	// Verify that the context returned from the handler is also returned from Start,
	// and is the context passed to End.
	ctx := event.WithExporter(context.Background(), event.NewExporter(&testTraceHandler{t: t}, eventtest.ExporterOptions()))
	ctx = event.Start(ctx, "s")
	val := ctx.Value("x")
	if val != 1 {
		t.Fatal("context not returned from Start")
	}
	event.End(ctx)
}

type testTraceHandler struct {
	t *testing.T
}

func (t *testTraceHandler) Event(ctx context.Context, ev *event.Event) context.Context {
	switch ev.Kind {
	case event.StartKind:
		return context.WithValue(ctx, "x", 1)
	case event.EndKind:
		val := ctx.Value("x")
		if val != 1 {
			t.t.Fatal("Start context not passed to End")
		}
		return ctx
	default:
		return ctx
	}
}

func TestTraceDuration(t *testing.T) {
	// Verify that a trace can can emit a latency metric.
	dur := event.NewDuration("test", "")
	want := time.Second

	check := func(t *testing.T, h *testTraceDurationHandler) {
		if !h.got.HasValue() {
			t.Fatal("no metric value")
		}
		got := h.got.Duration()
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}

	t.Run("returned builder", func(t *testing.T) {
		h := &testTraceDurationHandler{}
		ctx := event.WithExporter(context.Background(), event.NewExporter(h, eventtest.ExporterOptions()))
		ctx = event.Start(ctx, "s")
		time.Sleep(want)
		event.End(ctx, event.DurationMetric.Of(dur))
		check(t, h)
	})
	//TODO: come back and fix this
	t.Run("separate builder", func(t *testing.T) {
		h := &testTraceDurationHandler{}
		ctx := event.WithExporter(context.Background(), event.NewExporter(h, eventtest.ExporterOptions()))
		ctx = event.Start(ctx, "s")
		time.Sleep(want)
		event.End(ctx, event.DurationMetric.Of(dur))
		check(t, h)
	})
}

type testTraceDurationHandler struct {
	got event.Label
}

func (t *testTraceDurationHandler) Event(ctx context.Context, ev *event.Event) context.Context {
	for _, l := range ev.Labels {
		if l.Name == event.MetricVal {
			t.got = l
		}
	}
	return ctx
}

func BenchmarkBuildContext(b *testing.B) {
	// How long does it take to deliver an event from a nested context?
	c := event.NewCounter("c", "")
	for _, depth := range []int{1, 5, 7, 10} {
		b.Run(fmt.Sprintf("depth %d", depth), func(b *testing.B) {
			ctx := event.WithExporter(context.Background(), event.NewExporter(nopHandler{}, eventtest.ExporterOptions()))
			for i := 0; i < depth; i++ {
				ctx = context.WithValue(ctx, i, i)
			}
			b.Run("direct", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					c.Record(ctx, 1)
				}
			})
			/*TODO: work out how we do cached labels
			b.Run("cloned", func(b *testing.B) {
				bu := event.To(ctx)
				for i := 0; i < b.N; i++ {
					c.RecordTB(bu, 1).Name("foo").Send()
				}
			})
			*/
		})
	}
}
