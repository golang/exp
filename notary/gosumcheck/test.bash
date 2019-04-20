#!/bin/bash

set -e
go build -o gosumcheck.exe
export GONOVERIFY=*/text # rsc.io/text but not golang.org/x/text
./gosumcheck.exe "$@" -v test.sum
rm -f ./gosumcheck.exe
