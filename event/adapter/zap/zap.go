// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

// zap provides an implementation of zapcore.Core for events.
// To use globally:
//     zap.ReplaceGlobals(zap.New(NewCore(exporter)))
//
// If you call elogging.SetExporter, then you can pass nil
// for the exporter above and it will use the global one.
package zap

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/severity"
)

type core struct {
	ev     *event.Event // cloned but never delivered
	labels []event.Label
}

var _ zapcore.Core = (*core)(nil)

func NewCore(ctx context.Context) zapcore.Core {
	return &core{ev: event.New(ctx, event.LogKind)}
}

func (c *core) Enabled(level zapcore.Level) bool {
	return true
}

func (c *core) With(fields []zapcore.Field) zapcore.Core {
	c2 := *c
	if len(fields) > 0 {
		c2.labels = make([]event.Label, len(c.labels), len(c.labels)+len(fields))
		copy(c2.labels, c.labels)
		for _, f := range fields {
			c2.labels = append(c2.labels, newLabel(f))
		}
	}
	return &c2
}

func (c *core) Write(e zapcore.Entry, fs []zapcore.Field) error {
	if c.ev == nil {
		return nil
	}
	ev := c.ev.Clone()
	if ev == nil {
		return nil
	}
	ev.At = e.Time
	// TODO: Add labels more efficiently: compare cap(ev.Labels) to the total number to add,
	// and allocate a []Label once.
	ev.Labels = append(ev.Labels, c.labels...)
	ev.Labels = append(ev.Labels, convertLevel(e.Level).Label())
	ev.Labels = append(ev.Labels, event.String("name", e.LoggerName))
	if e.Stack != "" {
		ev.Labels = append(ev.Labels, event.String("stack", e.Stack))
	}
	if e.Caller.Defined {
		ev.Labels = append(ev.Labels, event.String("caller", e.Caller.String()))
	}
	for _, f := range fs {
		ev.Labels = append(ev.Labels, newLabel(f))
	}
	ev.Labels = append(ev.Labels, event.String("msg", e.Message))
	ev.Deliver()
	return nil
}

func (c *core) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(e, c)
}

func (c *core) Sync() error { return nil }

func newLabel(f zap.Field) event.Label {
	switch f.Type {
	case zapcore.ArrayMarshalerType, zapcore.ObjectMarshalerType, zapcore.BinaryType, zapcore.ByteStringType,
		zapcore.Complex128Type, zapcore.Complex64Type, zapcore.TimeFullType, zapcore.ReflectType,
		zapcore.ErrorType:
		return event.Value(f.Key, f.Interface)
	case zapcore.DurationType:
		return event.Duration(f.Key, time.Duration(f.Integer))
	case zapcore.Float64Type:
		return event.Float64(f.Key, math.Float64frombits(uint64(f.Integer)))
	case zapcore.Float32Type:
		return event.Float64(f.Key, float64(math.Float32frombits(uint32(f.Integer))))
	case zapcore.BoolType:
		b := false
		if f.Integer != 0 {
			b = true
		}
		return event.Bool(f.Key, b)
	case zapcore.Int64Type:
		return event.Int64(f.Key, f.Integer)
	case zapcore.Int32Type:
		return event.Int64(f.Key, f.Integer)

		//, zapcore.Int16Type, zapcore.Int8Type,
		// 		zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type, zapcore.UintptrType:
		//		return (f.Key,uint64(f.Integer))
	case zapcore.StringType:
		return event.String(f.Key, f.String)
	case zapcore.TimeType:
		t := time.Unix(0, f.Integer)
		if f.Interface != nil {
			t = t.In(f.Interface.(*time.Location))
		}
		return event.Value(f.Key, t)
	case zapcore.StringerType:
		return event.String(f.Key, stringerToString(f.Interface))
	case zapcore.NamespaceType:
		// TODO: ???
		return event.Label{}
	case zapcore.SkipType:
		// TODO: avoid creating a label at all in this case.
		return event.Label{}
	default:
		panic(fmt.Sprintf("unknown field type: %v", f))
	}
}

// Adapter from encodeStringer in go.uber.org/zap/zapcore/field.go.
func stringerToString(stringer interface{}) (s string) {
	// Try to capture panics (from nil references or otherwise) when calling
	// the String() method, similar to https://golang.org/src/fmt/print.go#L540
	defer func() {
		if err := recover(); err != nil {
			// If it's a nil pointer, just say "<nil>". The likeliest causes are a
			// Stringer that fails to guard against nil or a nil pointer for a
			// value receiver, and in either case, "<nil>" is a nice result.
			if v := reflect.ValueOf(stringer); v.Kind() == reflect.Ptr && v.IsNil() {
				s = "<nil>"
				return
			}
			s = fmt.Sprintf("PANIC=%v", err)
		}
	}()

	return stringer.(fmt.Stringer).String()
}

func convertLevel(level zapcore.Level) severity.Level {
	switch level {
	case zapcore.DebugLevel:
		return severity.Debug
	case zapcore.InfoLevel:
		return severity.Info
	case zapcore.WarnLevel:
		return severity.Warning
	case zapcore.ErrorLevel:
		return severity.Error
	case zapcore.DPanicLevel:
		return severity.Fatal - 1
	case zapcore.PanicLevel:
		return severity.Fatal + 1
	case zapcore.FatalLevel:
		return severity.Fatal
	default:
		return severity.Trace
	}
}
