// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
)

func TestClone(t *testing.T) {
	var labels []event.Label
	for i := 0; i < 5; i++ { // one greater than len(Builder.labels)
		labels = append(labels, keys.Int(fmt.Sprintf("l%d", i)).Of(i))
	}

	ctx := event.WithExporter(context.Background(), event.NewExporter(event.NopHandler{}, nil))
	b1 := event.To(ctx)
	b1.With(labels[0]).With(labels[1])
	check(t, b1, labels[:2])
	b2 := b1.Clone()
	check(t, b1, labels[:2])
	check(t, b2, labels[:2])

	b2.With(labels[2])
	check(t, b1, labels[:2])
	check(t, b2, labels[:3])

	// Force a new backing array for b.Event.Labels.
	for i := 3; i < len(labels); i++ {
		b2.With(labels[i])
	}
	check(t, b1, labels[:2])
	check(t, b2, labels)

	b2.Log("") // put b2 back in the pool.
	b2 = event.To(ctx)
	check(t, b1, labels[:2])
	check(t, b2, []event.Label{})

	b2.With(labels[3]).With(labels[4])
	check(t, b1, labels[:2])
	check(t, b2, labels[3:5])
}

func check(t *testing.T, b event.Builder, want []event.Label) {
	t.Helper()
	if got := b.Event().Labels; !cmp.Equal(got, want, cmp.Comparer(valueEqual)) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func valueEqual(l1, l2 event.Value) bool {
	return fmt.Sprint(l1) == fmt.Sprint(l2)
}

func TestTraceBuilder(t *testing.T) {
	// Verify that the context returned from the handler is also returned from Start,
	// and is the context passed to End.
	ctx := event.WithExporter(context.Background(), event.NewExporter(&testTraceHandler{t: t}, nil))
	ctx, end := event.To(ctx).Start("s")
	val := ctx.Value("x")
	if val != 1 {
		t.Fatal("context not returned from Start")
	}
	end()
}

type testTraceHandler struct {
	event.NopHandler
	t *testing.T
}

func (*testTraceHandler) Start(ctx context.Context, _ *event.Event) context.Context {
	return context.WithValue(ctx, "x", 1)
}

func (t *testTraceHandler) End(ctx context.Context, _ *event.Event) {
	val := ctx.Value("x")
	if val != 1 {
		t.t.Fatal("Start context not passed to End")
	}
}

func TestFailToClone(t *testing.T) {
	ctx := event.WithExporter(context.Background(), event.NewExporter(event.NopHandler{}, nil))

	catch := func(f func()) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic, did not get one")
				return
			}
			got, ok := r.(string)
			if !ok || !strings.Contains(got, "Clone") {
				t.Errorf("got panic(%v), want string with 'Clone'", r)
			}
		}()

		f()
	}

	catch(func() {
		b1 := event.To(ctx)
		b1.Log("msg1")
		// Reuse of Builder without Clone; b1.data has been cleared.
		b1.Log("msg2")
	})

	catch(func() {
		b1 := event.To(ctx)
		b1.Log("msg1")
		_ = event.To(ctx) // re-allocate the builder
		// b1.data is populated, but with the wrong information.
		b1.Log("msg2")
	})
}

func TestTraceDuration(t *testing.T) {
	// Verify that a trace can can emit a latency metric.
	dur := event.NewDuration("test", "")
	want := 200 * time.Millisecond

	check := func(t *testing.T, h *testTraceDurationHandler) {
		if !h.got.HasValue() {
			t.Fatal("no metric value")
		}
		got := h.got.Duration().Round(50 * time.Millisecond)
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}

	t.Run("end function", func(t *testing.T) {
		h := &testTraceDurationHandler{}
		ctx := event.WithExporter(context.Background(), event.NewExporter(h, nil))
		ctx, end := event.To(ctx).With(event.DurationMetric.Of(dur)).Start("s")
		time.Sleep(want)
		end()
		check(t, h)
	})
	t.Run("End method", func(t *testing.T) {
		h := &testTraceDurationHandler{}
		ctx := event.WithExporter(context.Background(), event.NewExporter(h, nil))
		ctx, _ = event.To(ctx).Start("s")
		time.Sleep(want)
		event.To(ctx).With(event.DurationMetric.Of(dur)).End()
		check(t, h)
	})
}

type testTraceDurationHandler struct {
	event.NopHandler
	got event.Value
}

func (t *testTraceDurationHandler) Metric(ctx context.Context, e *event.Event) {
	t.got, _ = event.MetricVal.Find(e)
}

func BenchmarkBuildContext(b *testing.B) {
	// How long does it take to deliver an event from a nested context?
	c := event.NewCounter("c", "")
	for _, depth := range []int{1, 5, 7, 10} {
		b.Run(fmt.Sprintf("depth %d", depth), func(b *testing.B) {
			ctx := event.WithExporter(context.Background(), event.NewExporter(event.NopHandler{}, nil))
			for i := 0; i < depth; i++ {
				ctx = context.WithValue(ctx, i, i)
			}
			b.Run("direct", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					event.To(ctx).With(event.Name.Of("foo")).Metric(c.Record(1))
				}
			})
			b.Run("cloned", func(b *testing.B) {
				bu := event.To(ctx)
				for i := 0; i < b.N; i++ {
					bu.Clone().With(event.Name.Of("foo")).Metric(c.Record(1))
				}
			})
		})
	}
}
