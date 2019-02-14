#!/bin/bash

set -e
go build -o notecheck.exe
./notecheck.exe "$@" -v rsc-goog.appspot.com+eecb1dec+AbTy1QXWdqYd1TTpuaUqsk6u7p+n4AqLiLB8SBwoB831 test.sum
rm -f ./notecheck.exe
