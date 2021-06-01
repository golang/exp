// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package elogrus

import (
	"context"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
	"golang.org/x/exp/event/logging/internal"
)

func Test(t *testing.T) {
	e, th := internal.NewTestExporter()
	log := logrus.New()
	log.SetFormatter(NewFormatter())
	log.SetOutput(io.Discard)
	// adding WithContext panics, because event.FromContext assumes
	ctx := event.WithExporter(context.Background(), e)
	log.WithContext(ctx).WithField("traceID", 17).WithField("resource", "R").Info("mess")

	want := &event.Event{
		Labels: []event.Label{
			internal.LevelKey.Of(4),
			keys.Value("traceID").Of(17),
			keys.Value("resource").Of("R"),
			event.Message.Of("mess"),
		},
	}
	th.Got.At = want.At
	// logrus fields are stored in a map, so we have to sort to overcome map
	// iteration indeterminacy.
	less := func(a, b event.Label) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want, &th.Got, append([]cmp.Option{cmpopts.SortSlices(less)})...); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}

}
