#!/bin/bash -e

go=${1:-go}

cd $(dirname $0)

set -x

# Run all benchmarks a few times and capture to a file.
$go test -bench . -count 5 > zerolog.bench

# Rename the package in the output to fool benchstat into comparing
# these benchmarks with the ones in the parent directory.
sed -i -e 's?^pkg: .*$?pkg: golang.org/x/exp/slog/benchmarks?' zerolog.bench
