// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package severity

import "golang.org/x/exp/event"

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
	TraceLevel   = Level(1)
	DebugLevel   = Level(5)
	InfoLevel    = Level(9)
	WarningLevel = Level(13)
	ErrorLevel   = Level(17)
	FatalLevel   = Level(21)
	MaxLevel     = Level(24)
)

const Key = "level"

var (
	// Trace is an event.Label for trace level events.
	Trace = Of(TraceLevel)
	// Debug is an event.Label for debug level events.
	Debug = Of(DebugLevel)
	// Info is an event.Label for info level events.
	Info = Of(InfoLevel)
	// Warning is an event.Label for warning level events.
	Warning = Of(WarningLevel)
	// Error is an event.Label for error level events.
	Error = Of(ErrorLevel)
	// Fatal is an event.Label for fatal level events.
	Fatal = Of(FatalLevel)
)

// Of creates a new Label with this key and the supplied value.
func Of(v Level) event.Label {
	return event.Label{Name: Key, Value: event.ValueOf(v)}
}

// From can be used to get a value from a Label.
func From(t event.Label) Level {
	return t.Value.Interface().(Level)
}

func (l Level) Class() Level {
	switch {
	case l > MaxLevel:
		return MaxLevel
	case l > FatalLevel:
		return FatalLevel
	case l > ErrorLevel:
		return ErrorLevel
	case l > WarningLevel:
		return WarningLevel
	case l > DebugLevel:
		return DebugLevel
	case l > InfoLevel:
		return InfoLevel
	case l > TraceLevel:
		return TraceLevel
	default:
		return 0
	}
}

func (l Level) String() string {
	switch l {
	case 0:
		return "invalid"

	case TraceLevel:
		return "trace"
	case TraceLevel + 1:
		return "trace2"
	case TraceLevel + 2:
		return "trace3"
	case TraceLevel + 3:
		return "trace4"

	case DebugLevel:
		return "debug"
	case DebugLevel + 1:
		return "debug2"
	case DebugLevel + 2:
		return "debug3"
	case DebugLevel + 3:
		return "debug4"

	case InfoLevel:
		return "info"
	case InfoLevel + 1:
		return "info2"
	case InfoLevel + 2:
		return "info3"
	case InfoLevel + 3:
		return "info4"

	case WarningLevel:
		return "warning"
	case WarningLevel + 1:
		return "warning2"
	case WarningLevel + 2:
		return "warning3"
	case WarningLevel + 3:
		return "warning4"

	case ErrorLevel:
		return "error"
	case ErrorLevel + 1:
		return "error2"
	case ErrorLevel + 2:
		return "error3"
	case ErrorLevel + 3:
		return "error4"

	case FatalLevel:
		return "fatal"
	case FatalLevel + 1:
		return "fatal2"
	case FatalLevel + 2:
		return "fatal3"
	case FatalLevel + 3:
		return "fatal4"
	default:
		return "invalid"
	}
}
