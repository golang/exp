// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build tools
// +build tools

package main

import (
	_ "github.com/client9/misspell/cmd/misspell"
	_ "honnef.co/go/tools/cmd/staticcheck"
	_ "mvdan.cc/unparam"
)
