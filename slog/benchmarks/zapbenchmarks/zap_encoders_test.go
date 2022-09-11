// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zapbenchmarks

import (
	"fmt"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slog"
	slogbench "golang.org/x/exp/slog/benchmarks"
)

func TestEncoders(t *testing.T) {
	entry := zapcore.Entry{Time: slogbench.TestTime, Message: slogbench.TestMessage}
	fields := attrsToFields(slogbench.TestAttrs)
	t.Run("text", func(t *testing.T) {
		te := newFastTextEncoder()
		defer tePool.Put(te)
		buf, err := te.EncodeEntry(entry, fields)
		if err != nil {
			t.Fatal(err)
		}
		defer buf.Free()
		got := strings.ToLower(buf.String())
		want := strings.ToLower(slogbench.WantText)
		if got != want {
			t.Errorf("\ngot  %s\nwant %s", got, want)
		}
	})
}

func attrsToFields(attrs []slog.Attr) []zap.Field {
	var fields []zap.Field
	for _, a := range slogbench.TestAttrs {
		var f zap.Field
		k := a.Key()
		switch a.Kind() {
		case slog.StringKind:
			f = zap.String(k, a.String())
		case slog.Int64Kind:
			f = zap.Int64(k, a.Int64())
		case slog.DurationKind:
			f = zap.Duration(k, a.Duration())
		case slog.TimeKind:
			f = zap.Time(k, a.Time())
		case slog.AnyKind:
			f = zap.Any(k, a.Value())
		default:
			panic(fmt.Sprintf("unknown kind %d", a.Kind()))
		}
		fields = append(fields, f)
	}
	return fields
}
