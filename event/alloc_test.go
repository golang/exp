// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !race

package event_test

import (
	"context"
	"io/ioutil"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/eventtest"
)

func TestAllocs(t *testing.T) {
	anInt := event.Label{Name: "int", Value: event.Int64Of(4)}
	aString := event.Label{Name: "string", Value: event.StringOf("value")}

	e := event.NewExporter(logfmt.NewHandler(ioutil.Discard), nil)
	ctx := event.WithExporter(context.Background(), e)
	allocs := int(testing.AllocsPerRun(5, func() {
		event.To(ctx).With(aString).With(anInt).Log("message")
	}))
	if allocs != 0 {
		t.Errorf("Got %d allocs, expect 0", allocs)
	}
}

func TestBenchAllocs(t *testing.T) {
	eventtest.TestAllocs(t, eventPrint, eventLog, 0)
}
