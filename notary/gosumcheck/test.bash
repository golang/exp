#!/bin/bash

set -e
go build -o gosumcheck.exe
./gosumcheck.exe "$@" -v test.sum
rm -f ./gosumcheck.exe
