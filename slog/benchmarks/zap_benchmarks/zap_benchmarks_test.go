// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zap_benchmarks

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	slogbench "golang.org/x/exp/slog/benchmarks"
)

// Keep in sync (same names and behavior) as the
// benchmarks in the parent directory.

func BenchmarkAttrs(b *testing.B) {
	for _, logger := range []struct {
		name string
		l    *zap.Logger
	}{
		{
			"async discard",
			zap.New(&asyncCore{}),
		},
		{
			"fastText discard",
			zap.New(zapcore.NewCore(
				&fastTextEncoder{},
				&discarder{},
				zap.DebugLevel)),
		},
		{
			"JSON discard",
			zap.New(zapcore.NewCore(
				zapcore.NewJSONEncoder(zapcore.EncoderConfig{
					MessageKey: "msg",
					LevelKey:   "level",
					TimeKey:    "time",
					EncodeTime: zapcore.TimeEncoderOfLayout(time.RFC3339Nano),
				}),
				&discarder{},
				zap.DebugLevel)),
		}} {
		l := logger.l
		b.Run(logger.name, func(b *testing.B) {
			for _, call := range []struct {
				name string
				f    func()
			}{
				{
					"5 args",
					func() {
						l.Info(slogbench.TestMessage,
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError))
					},
				},
				{
					"10 args",
					func() {
						l.Info(slogbench.TestMessage,
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError),
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError))
					},
				},
				{
					"40 args",
					func() {
						l.Info(slogbench.TestMessage,
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError),
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError),
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError),
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError),
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError),
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError),
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError),
							zap.String("string", slogbench.TestString),
							zap.Int("status", slogbench.TestInt),
							zap.Duration("duration", slogbench.TestDuration),
							zap.Time("time", slogbench.TestTime),
							zap.Any("error", slogbench.TestError))
					},
				},
			} {
				b.Run(call.name, func(b *testing.B) {
					b.ReportAllocs()
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							call.f()
						}
					})
				})
			}
		})
	}
}
