// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
)

// TODO: test WithAll (which will currently break the aliasing check).

type testHandler struct{}

func (testHandler) Handle(*event.Event) {}

func TestClone(t *testing.T) {
	var labels []event.Label
	for i := 0; i < 5; i++ { // one greater than len(Builder.labels)
		labels = append(labels, keys.Int(fmt.Sprintf("l%d", i)).Of(i))
	}

	check := func(b *event.Builder, want []event.Label) {
		t.Helper()
		if got := b.Event.Labels; !cmp.Equal(got, want, cmp.Comparer(labelEqual)) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}

	e := event.NewExporter(testHandler{})
	b1 := e.Builder()
	b1.With(labels[0]).With(labels[1])
	check(b1, labels[:2])
	b2 := b1.Clone()
	check(b1, labels[:2])
	check(b2, labels[:2])

	b2.With(labels[2])
	check(b1, labels[:2])
	check(b2, labels[:3])

	// Force a new backing array for b.Event.Labels.
	for i := 3; i < len(labels); i++ {
		b2.With(labels[i])
	}
	check(b1, labels[:2])
	check(b2, labels)

	b2.Log("") // put b2 back in the pool.
	b2 = e.Builder()
	check(b1, labels[:2])
	check(b2, []event.Label{})

	b2.With(labels[3]).With(labels[4])
	check(b1, labels[:2])
	check(b2, labels[3:5])
}

func labelEqual(l1, l2 event.Label) bool {
	return labelString(l1) == labelString(l2)
}

func labelString(l event.Label) string {
	var buf bytes.Buffer
	p := event.NewPrinter(&buf)
	p.Label(l)
	return buf.String()
}
