#!/bin/bash

if [ "$1" == "test" ]; then
    go test ./...
    exit 0
fi

if [ "$1" == "test-n-cover" ]; then
    go test -coverprofile=/tmp/coverage.out ./... | perl -MFile::Slurp=edit_file -nWE 'print $_; next unless /coverage: ([0-9\.]+)%/; my$cov=$1; my$clr=$cov>70?"green":$cov>50?"yellow":"red"; edit_file {s/(\!\[\]\(https:\/\/img\.shields\.io\/static\/v1\?label=Coverage)&message=[0-9\.]+%&color=[a-z]+\)/$1&message=$cov%&color=$clr)/} "README.md"' 2>/dev/null
    exit 0
fi

if [ "$1" == "cover-view" ]; then
    go tool cover -html=/tmp/coverage.out
    exit 0
fi
