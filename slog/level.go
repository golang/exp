// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"fmt"
	"math"
	"strconv"
	"sync/atomic"
)

// A Level is the importance or severity of a log event.
// The higher the level, the less important or severe the event.
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
// The lack of a gap between Debug and Info doesn't follow the pattern.
// It makes sense, though, that the first negative number is the start
// of the Debug range.
//
// Names for common levels.
const (
	DebugLevel Level = -1
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
	case l <= DebugLevel:
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

// An AtomicLevel is Level that can be read and written safely by multiple
// goroutines.
// Use NewAtomicLevel to create one.
type AtomicLevel struct {
	val atomic.Int64
}

// NewAtomicLevel creates an AtomicLevel initialized to the given Level.
func NewAtomicLevel(l Level) *AtomicLevel {
	var r AtomicLevel
	r.Set(l)
	return &r
}

// Level returns r's level.
// If r is nil, it returns the maximum level.
func (r *AtomicLevel) Level() Level {
	if r == nil {
		return Level(math.MaxInt)
	}
	return Level(int(r.val.Load()))
}

// Set sets r's level to l.
func (r *AtomicLevel) Set(l Level) {
	r.val.Store(int64(l))
}
