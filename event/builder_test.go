// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"context"
	"fmt"
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

	ctx := event.WithExporter(context.Background(), event.NewExporter(nil))
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
