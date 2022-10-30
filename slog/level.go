// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"fmt"
	"strconv"
	"sync/atomic"
)

// A Level is the importance or severity of a log event.
// The higher the level, the more important or severe the event.
type Level int

// The level numbers below don't really matter too much. Any system can map them
// to another numbering scheme if it wishes. We picked them to satisfy three
// constraints.
//
// First, we wanted the default level to be Info, Since Levels are ints, Info is
// the default value for int, zero.
//
// Second, we wanted to make it easy to work with verbosities instead of levels.
// Verbosities start at 0 corresponding to Info, and larger values are less severe
// Negating a verbosity converts it into a Level.
//
// Third, we wanted some room between levels to accommodate schemes with named
// levels between ours. For example, Google Cloud Logging defines a Notice level
// between Info and Warn. Since there are only a few of these intermediate
// levels, the gap between the numbers need not be large. Our gap of 4 matches
// OpenTelemetry's mapping. Subtracting 9 from an OpenTelemetry level in the
// DEBUG, INFO, WARN and ERROR ranges converts it to the corresponding slog
// Level range. OpenTelemetry also has the names TRACE and FATAL, which slog
// does not. But those OpenTelemetry levels can still be represented as slog
// Levels by using the appropriate integers.
//
// Names for common levels.
const (
	DebugLevel Level = -4
	InfoLevel  Level = 0
	WarnLevel  Level = 4
	ErrorLevel Level = 8
)

// String returns a name for the level.
// If the level has a name, then that name
// in uppercase is returned.
// If the level is between named values, then
// an integer is appended to the uppercased name.
// Examples:
//
//	WarnLevel.String() => "WARN"
//	(WarnLevel-2).String() => "WARN-2"
func (l Level) String() string {
	str := func(base string, val Level) string {
		if val == 0 {
			return base
		}
		return fmt.Sprintf("%s%+d", base, val)
	}

	switch {
	case l < InfoLevel:
		return str("DEBUG", l-DebugLevel)
	case l < WarnLevel:
		return str("INFO", l)
	case l < ErrorLevel:
		return str("WARN", l-WarnLevel)
	default:
		return str("ERROR", l-ErrorLevel)
	}
}

func (l Level) MarshalJSON() ([]byte, error) {
	// AppendQuote is sufficient for JSON-encoding all Level strings.
	// They don't contain any runes that would produce invalid JSON
	// when escaped.
	return strconv.AppendQuote(nil, l.String()), nil
}

// Level returns the receiver.
// It implements Leveler.
func (l Level) Level() Level { return l }

// A LevelVar is a Level variable, to allow a Handler level to change
// dynamically.
// It implements Leveler as well as a Set method,
// and it is safe for use by multiple goroutines.
// The zero LevelVar corresponds to InfoLevel.
type LevelVar struct {
	val atomic.Int64
}

// Level returns v's level.
func (v *LevelVar) Level() Level {
	return Level(int(v.val.Load()))
}

// Set sets v's level to l.
func (v *LevelVar) Set(l Level) {
	v.val.Store(int64(l))
}

func (v *LevelVar) String() string {
	return fmt.Sprintf("LevelVar(%s)", v.Level())
}

// A Leveler provides a Level value.
//
// As Level itself implements Leveler, clients typically supply
// a Level value wherever a Leveler is needed, such as in HandlerOptions.
// Clients who need to vary the level dynamically can provide a more complex
// Leveler implementation such as *LevelVar.
type Leveler interface {
	Level() Level
}
