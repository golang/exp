// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package severity

import (
	"context"
	"fmt"

	"golang.org/x/exp/event"
)

// Level represents a severity level of an event.
// The basic severity levels are designed to match the levels used in open telemetry.
// Smaller numerical values correspond to less severe events (such as debug events),
// larger numerical values correspond to more severe events (such as errors and critical events).
//
// The following table defines the meaning severity levels:
// 1-4	TRACE	A fine-grained debugging event. Typically disabled in default configurations.
// 5-8	DEBUG	A debugging event.
// 9-12	INFO	An informational event. Indicates that an event happened.
// 13-16	WARN	A warning event. Not an error but is likely more important than an informational event.
// 17-20	ERROR	An error event. Something went wrong.
// 21-24	FATAL	A fatal error such as application or system crash.
//
// See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/data-model.md#severity-fields
// for more details
type Level uint64

const (
	Trace   = Level(1)
	Debug   = Level(5)
	Info    = Level(9)
	Warning = Level(13)
	Error   = Level(17)
	Fatal   = Level(21)
	Max     = Level(24)
)

const Key = "level"

// Of creates a label for the level.
func (l Level) Label() event.Label {
	return event.Value(Key, l)
}

// From can be used to get a value from a Label.
func From(t event.Label) Level {
	return t.Interface().(Level)
}

func (l Level) Log(ctx context.Context, msg string, labels ...event.Label) {
	ev := event.New(ctx, event.LogKind)
	if ev != nil {
		ev.Labels = append(ev.Labels, l.Label())
		ev.Labels = append(ev.Labels, labels...)
		ev.Labels = append(ev.Labels, event.String("msg", msg))
		ev.Deliver()
	}
}

func (l Level) Logf(ctx context.Context, msg string, args ...interface{}) {
	ev := event.New(ctx, event.LogKind)
	if ev != nil {
		ev.Labels = append(ev.Labels, l.Label())
		ev.Labels = append(ev.Labels, event.String("msg", fmt.Sprintf(msg, args...)))
		ev.Deliver()
	}
}

func (l Level) Class() Level {
	switch {
	case l > Max:
		return Max
	case l > Fatal:
		return Fatal
	case l > Error:
		return Error
	case l > Warning:
		return Warning
	case l > Debug:
		return Debug
	case l > Info:
		return Info
	case l > Trace:
		return Trace
	default:
		return 0
	}
}

func (l Level) String() string {
	switch l {
	case 0:
		return "invalid"

	case Trace:
		return "trace"
	case Trace + 1:
		return "trace2"
	case Trace + 2:
		return "trace3"
	case Trace + 3:
		return "trace4"

	case Debug:
		return "debug"
	case Debug + 1:
		return "debug2"
	case Debug + 2:
		return "debug3"
	case Debug + 3:
		return "debug4"

	case Info:
		return "info"
	case Info + 1:
		return "info2"
	case Info + 2:
		return "info3"
	case Info + 3:
		return "info4"

	case Warning:
		return "warning"
	case Warning + 1:
		return "warning2"
	case Warning + 2:
		return "warning3"
	case Warning + 3:
		return "warning4"

	case Error:
		return "error"
	case Error + 1:
		return "error2"
	case Error + 2:
		return "error3"
	case Error + 3:
		return "error4"

	case Fatal:
		return "fatal"
	case Fatal + 1:
		return "fatal2"
	case Fatal + 2:
		return "fatal3"
	case Fatal + 3:
		return "fatal4"
	default:
		return "invalid"
	}
}
