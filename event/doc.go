// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package event provides low-cost tracing, metrics and structured logging.
// These are often grouped under the term "observability".
//
// This package is highly experimental and in a state of flux; it is public only
// to allow for collaboration on the design and implementation, and may be
// changed or removed at any time.
//
// It uses a common event system to provide a way for libraries to produce
// observability information in a way that does not tie the libraries to a
// specific API or applications to a specific export format.
//
// It is designed for minimal overhead when no exporter is used so that it is
// safe to leave calls in libraries.
package event
