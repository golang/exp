#!/bin/bash -e

go=${1:-go}

cd $(dirname $0)

set -x

$go test            -bench . -count 5 > slog.bench
$go test -tags nopc -bench . -count 5 > slog-nopc.bench
