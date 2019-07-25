#!/bin/bash

function @execute() { echo "$@" >&2; "$@"; }

set -e

"$(dirname "$0")/docker/build.sh"
@execute docker tag server asia.gcr.io/prog-edu-assistant/server
@execute docker tag worker asia.gcr.io/prog-edu-assistant/worker
@execute docker push asia.gcr.io/prog-edu-assistant/server
@execute docker push asia.gcr.io/prog-edu-assistant/worker

echo OK
