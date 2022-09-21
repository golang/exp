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
// to another numbering scheme if it wishes. We picked them to satisfy two
// constraints.
//
// First, we wanted to make it easy to work with verbosities instead of levels.
// Since higher verbosities are less important, higher levels are as well.
//
// Second, we wanted some room between levels to accommodate schemes with named
// levels between ours. For example, Google Cloud Logging defines a Notice level
// between Info and Warn. Since there are only a few of these intermediate
// levels, the gap between the numbers need not be large. We selected a gap of
// 10, because the majority of humans have 10 fingers.
//
// The missing gap between Info and Debug has to do with verbosities again. It
// is natural to think of verbosity 0 as Info, and then verbosity 1 is the
// lowest level one would call Debug. The simple formula
//   level = InfoLevel + verbosity
// then works well to map verbosities to levels. That is,
//
//   Level(InfoLevel+0).String() == "INFO"
//   Level(InfoLevel+1).String() == "DEBUG"
//   Level(InfoLevel+2).String() == "DEBUG+1"
//
// and so on.

// Names for common levels.
const (
	ErrorLevel Level = 10
	WarnLevel  Level = 20
	InfoLevel  Level = 30
	DebugLevel Level = 31
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
	case l <= 0:
		return fmt.Sprintf("!BADLEVEL(%d)", l)
	case l <= ErrorLevel:
		return str("ERROR", l-ErrorLevel)
	case l <= WarnLevel:
		return str("WARN", l-WarnLevel)
	case l <= InfoLevel:
		return str("INFO", l-InfoLevel)
	default:
		return str("DEBUG", l-DebugLevel)
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
