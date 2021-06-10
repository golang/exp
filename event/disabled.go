// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build disable_events

package event

import (
	"context"
	"time"
)

type Builder struct{ ctx context.Context }
type Exporter struct {
	Now func() time.Time
}

func NewExporter(h interface{}) *Exporter { return &Exporter{} }

func To(ctx context.Context) Builder                        { return Builder{} }
func (b Builder) Clone() Builder                            { return b }
func (b Builder) Label(label Label) Builder                 { return b }
func (b Builder) Labels(labels ...Label) Builder            { return b }
func (b Builder) Log(message string)                        {}
func (b Builder) Logf(template string, args ...interface{}) {}
func (b Builder) Metric()                                   {}
func (b Builder) Annotate()                                 {}
func (b Builder) End()                                      {}
func (b Builder) Event() *Event                             { return &Event{} }
func (b Builder) Start(name string) (context.Context, func()) {
	return b.ctx, func() {}
}

func newContext(ctx context.Context, exporter *Exporter, parent uint64) context.Context {
	return ctx
}
func FromContext(ctx context.Context) *Target { return nil }

func setDefaultExporter(e *Exporter) {}
