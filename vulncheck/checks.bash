#!/usr/bin/env bash
# Copyright 2021 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# This file will be run by `go test`.
# See all_test.go in this directory.

# Ensure that installed go binaries are on the path.
# This bash expression follows the algorithm described at the top of
# `go install help`: first try $GOBIN, then $GOPATH/bin, then $HOME/go/bin.
go_install_dir=${GOBIN:-${GOPATH:-$HOME/go}/bin}
PATH=$PATH:$go_install_dir

source ../devtools/checklib.sh

# check_unparam runs unparam on source files.
check_unparam() {
  ensure_go_binary mvdan.cc/unparam
  runcmd unparam ./...
}

# check_staticcheck runs staticcheck on source files.
check_staticcheck() {
  if [[ $(go version) = *go1.17* ]]; then
    ensure_go_binary honnef.co/go/tools/cmd/staticcheck
    runcmd staticcheck ./...
  fi
}

# check_misspell runs misspell on source files.
check_misspell() {
  ensure_go_binary github.com/client9/misspell/cmd/misspell
  runcmd misspell -error .
}

runchecks() {
  check_header *.go internal/*/*.go *.bash
  runcmd go vet -all ./...
  check_staticcheck
  check_unparam
  check_misspell
  runcmd go mod tidy
}

main() {
  runchecks
  if [[ $EXIT_CODE != 0 ]]; then
    err "FAILED; see errors above"
  fi
  exit $EXIT_CODE
}

main $@
