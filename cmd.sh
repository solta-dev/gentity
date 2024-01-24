#!/bin/bash

if [ "$1" == "test" ]; then
    go test ./...
    exit 0
fi

if [ "$1" == "test-n-cover" ]; then
    go test -coverprofile=cover.profile ./...
    exit 0
fi

if [ "$1" == "cover-view" ]; then
    go tool cover -html=cover.profile
    exit 0
fi
