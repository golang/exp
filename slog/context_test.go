// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"context"
	"testing"
)

func TestContext(t *testing.T) {
	// If there is no Logger in the context, FromContext returns the default
	// Logger.
	ctx := context.Background()
	gotl := FromContext(ctx)
	if _, ok := gotl.Handler().(*defaultHandler); !ok {
		t.Error("did not get default Logger")
	}

	// If there is a Logger in the context, FromContext returns it, with the ctx
	// arg.
	h := &captureHandler{}
	ctx = NewContext(ctx, New(h))
	ctx = context.WithValue(ctx, "ID", 1)
	gotl = FromContext(ctx)
	if gotl.Handler() != h {
		t.Fatal("wrong handler")
	}
	// FromContext preserves the context of the Logger that was stored
	// with NewContext, in this case nil.
	gotl.Info("")
	if g := h.r.Context; g != nil {
		t.Errorf("got %v, want nil", g)
	}
	gotl = Ctx(ctx)
	if gotl.Handler() != h {
		t.Fatal("wrong handler")
	}
	// Ctx adds the argument context to the Logger.
	gotl.Info("")
	c := h.r.Context
	if c == nil {
		t.Fatal("got nil Context")
	}
	if g, w := c.Value("ID"), 1; g != w {
		t.Errorf("got ID %v, want %v", g, w)
	}
}
