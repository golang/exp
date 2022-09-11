#!/bin/bash -ex

cd $(dirname $0)

# Run all benchmarks a few times and capture to a file.
go test -bench . -count 5 > zap.bench

# Rename the package in the output to fool benchstat into comparing
# these benchmarks with the ones in the parent directory.
sed -i -e 's?^pkg: .*$?pkg: golang.org/x/exp/slog/benchmarks?' zap.bench
