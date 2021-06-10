// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/keys"
)

var (
	l1      = keys.Int("l1").Of(1)
	l2      = keys.Int("l2").Of(2)
	l3      = keys.Int("l3").Of(3)
	counter = event.NewCounter("hits", "cache hits")
	gauge   = event.NewFloatGauge("temperature", "CPU board temperature in Celsius")
	latency = event.NewDuration("latency", "how long it took")
)

func TestPrint(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name   string
		events func(context.Context)
		expect string
	}{{
		name:   "simple",
		events: func(ctx context.Context) { event.Log(ctx, "a message") },
		expect: `time="2020/03/05 14:27:48" msg="a message"
`}, {
		name:   "log 1",
		events: func(ctx context.Context) { event.Log(ctx, "a message", l1) },
		expect: `time="2020/03/05 14:27:48" l1=1 msg="a message"`,
	}, {
		name:   "log 2",
		events: func(ctx context.Context) { event.Log(ctx, "a message", l1, l2) },
		expect: `time="2020/03/05 14:27:48" l1=1 l2=2 msg="a message"`,
	}, {
		name:   "log 3",
		events: func(ctx context.Context) { event.Log(ctx, "a message", l1, l2, l3) },
		expect: `time="2020/03/05 14:27:48" l1=1 l2=2 l3=3 msg="a message"`,
	}, {
		name: "span",
		events: func(ctx context.Context) {
			ctx = event.Start(ctx, "span")
			event.End(ctx)
		},
		expect: `
time="2020/03/05 14:27:48" trace=1 name=span
time="2020/03/05 14:27:49" parent=1 end
`}, {
		name: "span nested",
		events: func(ctx context.Context) {
			ctx = event.Start(ctx, "parent")
			defer event.End(ctx)
			child := event.Start(ctx, "child")
			defer event.End(child)
			event.Log(child, "message")
		},
		expect: `
time="2020/03/05 14:27:48" trace=1 name=parent
time="2020/03/05 14:27:49" parent=1 trace=2 name=child
time="2020/03/05 14:27:50" parent=2 msg=message
time="2020/03/05 14:27:51" parent=2 end
time="2020/03/05 14:27:52" parent=1 end
`}, {
		name:   "counter",
		events: func(ctx context.Context) { counter.Record(ctx, 2, l1) },
		expect: `time="2020/03/05 14:27:48" metricValue=2 metric="Metric(\"golang.org/x/exp/event_test/hits\")" l1=1`,
	}, {
		name:   "gauge",
		events: func(ctx context.Context) { gauge.Record(ctx, 98.6, l1) },
		expect: `time="2020/03/05 14:27:48" metricValue=98.6 metric="Metric(\"golang.org/x/exp/event_test/temperature\")" l1=1`,
	}, {
		name: "duration",
		events: func(ctx context.Context) {
			latency.Record(ctx, 3*time.Second, l1, l2)
		},
		expect: `time="2020/03/05 14:27:48" metricValue=3s metric="Metric(\"golang.org/x/exp/event_test/latency\")" l1=1 l2=2`,
	}, {
		name:   "annotate",
		events: func(ctx context.Context) { event.Annotate(ctx, l1) },
		expect: `time="2020/03/05 14:27:48" l1=1`,
	}, {
		name:   "annotate 2",
		events: func(ctx context.Context) { event.Annotate(ctx, l1, l2) },
		expect: `time="2020/03/05 14:27:48" l1=1 l2=2`,
	}, {
		name: "multiple events",
		events: func(ctx context.Context) {
			/*TODO: this is supposed to be using a cached target
			t := event.To(ctx)
			p := event.Prototype{}.As(event.LogKind)
			t.With(p).Int("myInt", 6).Message("my event").Send()
			t.With(p).String("myString", "some string value").Message("string event").Send()
			*/
			event.Log(ctx, "my event", keys.Int("myInt").Of(6))
			event.Log(ctx, "string event", keys.String("myString").Of("some string value"))
		},
		expect: `
time="2020/03/05 14:27:48" myInt=6 msg="my event"
time="2020/03/05 14:27:49" myString="some string value" msg="string event"
`}} {
		buf := &strings.Builder{}
		ctx := event.WithExporter(ctx, event.NewExporter(logfmt.NewHandler(buf), eventtest.ExporterOptions()))
		test.events(ctx)
		got := strings.TrimSpace(buf.String())
		expect := strings.TrimSpace(test.expect)
		if got != expect {
			t.Errorf("%s failed\ngot   : %s\nexpect: %s", test.name, got, expect)
		}
	}
}

func ExampleLog() {
	ctx := event.WithExporter(context.Background(), event.NewExporter(logfmt.NewHandler(os.Stdout), eventtest.ExporterOptions()))
	event.Log(ctx, "my event", keys.Int("myInt").Of(6))
	event.Log(ctx, "error event", keys.String("myString").Of("some string value"))
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
	got event.Value
}

func (t *testTraceDurationHandler) Event(ctx context.Context, ev *event.Event) context.Context {
	if ev.Kind == event.MetricKind {
		t.got, _ = event.MetricVal.Find(ev)
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
