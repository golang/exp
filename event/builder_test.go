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

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
)

func TestClone(t *testing.T) {
	var labels []event.Label
	for i := 0; i < 5; i++ { // one greater than len(Builder.labels)
		labels = append(labels, keys.Int(fmt.Sprintf("l%d", i)).Of(i))
	}

	ctx := event.WithExporter(context.Background(), event.NewExporter(noopHandler{}, nil))
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
	ctx := event.WithExporter(context.Background(), event.NewExporter(&testTraceHandler{t}, nil))
	ctx, end := event.To(ctx).Start("s")
	val := ctx.Value("x")
	if val != 1 {
		t.Fatal("context not returned from Start")
	}
	end()
}

type testTraceHandler struct {
	t *testing.T
}

func (*testTraceHandler) Log(ctx context.Context, _ *event.Event)      {}
func (*testTraceHandler) Annotate(ctx context.Context, _ *event.Event) {}
func (*testTraceHandler) Metric(ctx context.Context, _ *event.Event)   {}

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
	ctx := event.WithExporter(context.Background(), event.NewExporter(noopHandler{}, nil))

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

type noopHandler struct{}

func (noopHandler) Log(ctx context.Context, ev *event.Event)      {}
func (noopHandler) Metric(ctx context.Context, ev *event.Event)   {}
func (noopHandler) Annotate(ctx context.Context, ev *event.Event) {}
func (noopHandler) End(ctx context.Context, ev *event.Event)      {}
func (noopHandler) Start(ctx context.Context, ev *event.Event) context.Context {
	return ctx
}
