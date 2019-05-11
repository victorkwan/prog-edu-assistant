#!/bin/bash

function @execute() { echo "$@" >&2; "$@"; }

set -e

@execute mkdir -p bin
# Fully static builds.
CGO_ENABLED=0 GOOS=linux @execute go build -a -ldflags '-extldflags "-static"' -o bin/worker cmd/worker/worker.go
CGO_ENABLED=0 GOOS=linux @execute go build -a -ldflags '-extldflags "-static"' -o bin/server cmd/uploadserver/main.go

