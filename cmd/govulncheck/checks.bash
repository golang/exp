#!/usr/bin/env bash
# Copyright 2022 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

source ../../devtools/checklib.sh

# Support ** in globs for finding files throughout the tree.
shopt -s globstar

check_header **/*.go **/*.bash

set -x

go vet -all ./...
go run honnef.co/go/tools/cmd/staticcheck ./...
go run mvdan.cc/unparam ./...
go run github.com/client9/misspell/cmd/misspell -error ./...

go mod tidy
