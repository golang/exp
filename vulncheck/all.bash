#!/usr/bin/env bash
# Copyright 2021 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

if [[ $(basename $PWD) == "exp" ]]; then
  cd vulncheck
fi

RED=; GREEN=; YELLOW=; BLUE=; BOLD=; RESET=;

case $TERM in
  '' | xterm) ;;
  # If xterm is not xterm-16color, xterm-88color, or xterm-256color, tput will
  # return the error:
  #   tput: No value for $TERM and no -T specified
  *)
      RED=`tput setaf 1`
      GREEN=`tput setaf 2`
      YELLOW=`tput setaf 3`
      NORMAL=`tput sgr0`
esac

EXIT_CODE=0

info() { echo -e "${GREEN}$@${NORMAL}" 1>&2; }
err() { echo -e "${RED}$@${NORMAL}" 1>&2; EXIT_CODE=1; }

# runcud prints an info log describing the command that is about to be run, and
# then runs it. It sets EXIT_CODE to non-zero if the command fails, but does not exit
# the script.
runcmd() {
  # Truncate command logging for narrow terminals.
  # Account for the 2 characters of '$ '.
  maxwidth=$(( $(tput cols) - 2 ))
  if [[ ${#msg} -gt $maxwidth ]]; then
    msg="${msg::$(( maxwidth - 3 ))}..."
  fi

  echo -e "$@\n" 1>&2;
  $@ || err "command failed"
}

# ensure_go_binary verifies that a binary exists in $PATH corresponding to the
# given go-gettable URI. If no such binary exists, it is fetched via `go get`.
ensure_go_binary() {
  local binary=$(basename $1)
  if ! [ -x "$(command -v $binary)" ]; then
    info "Installing: $1"
    # Install the binary in a way that doesn't affect our go.mod file.
    go install $1@latest
  fi
}

# verify_header checks that all given files contain the standard header for Go
# projects.
verify_header() {
  if [[ "$@" != "" ]]; then
    for FILE in $@
    do
        line="$(head -1 $FILE)"
        if [[ ! $line == *"The Go Authors. All rights reserved."* ]] &&
         [[ ! $line == "// DO NOT EDIT. This file was copied from" ]]; then
              err "missing license header: $FILE"
        fi
    done
  fi
}

# Support ** in globs for findcode.
shopt -s globstar

# findcode finds source files in the repo.
findcode() {
  find **/*.go
}

# check_headers checks that all source files that have been staged in this
# commit, and all other non-third-party files in the repo, have a license
# header.
check_headers() {
  if [[ $# -gt 0 ]]; then
    info "Checking listed files for license header"
    verify_header $*
  else
    info "Checking staged files for license header"
    # Check code files that have been modified or added.
    verify_header $(git diff --cached --name-status | grep -vE "^D" | cut -f 2- | grep -E ".go$|.sh$")
    info "Checking go files for license header"
    verify_header $(findcode)
  fi
}

# check_unparam runs unparam on source files.
check_unparam() {
  ensure_go_binary mvdan.cc/unparam
  runcmd unparam ./...
}

# check_vet runs go vet on source files.
check_vet() {
  runcmd go vet -all ./...
}

# check_staticcheck runs staticcheck on source files.
check_staticcheck() {
  ensure_go_binary honnef.co/go/tools/cmd/staticcheck
  runcmd staticcheck ./...
}

# check_misspell runs misspell on source files.
check_misspell() {
  ensure_go_binary github.com/client9/misspell/cmd/misspell
  runcmd misspell -error .
}

go_linters() {
  check_vet
  check_staticcheck
  check_misspell
  check_unparam
}

go_modtidy() {
  runcmd go mod tidy
}

go_test() {
  runcmd go test ./...
}

runchecks() {
  check_headers
  go_linters
  go_modtidy
  go_test
}

usage() {
  cat <<EOUSAGE
Usage: $0 [subcommand]
Available subcommands:
  (empty)        - run all standard checks and tests:
     * headers: check source files for the license disclaimer
     * vet: run go vet on source files
     * staticcheck: run staticcheck on source files
     * misspell: run misspell on source files
     * unparam: run unparam on source files
     * tidy: run go mod tidy
     * tests: run all Go tests
  help           - display this help message
EOUSAGE
}

main() {
  case "$1" in
    "-h" | "--help" | "help")
      usage
      exit 0
      ;;
    "")
      runchecks
      ;;
    *)
      usage
      exit 1
  esac
  if [[ $EXIT_CODE != 0 ]]; then
    err "FAILED; see errors above"
  fi
  exit $EXIT_CODE
}

main $@
