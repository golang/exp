// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package event provides the core functionality for observability that allows
// libraries using it to interact well.
// It enforces the middle layer interchange format, but allows both frontend
// wrappers and back end exporters to customize the usage.
package event
