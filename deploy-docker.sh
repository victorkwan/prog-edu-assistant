#!/bin/bash

function @execute() { echo "$@" >&2; "$@"; }

GCE_HOST=${GCE_HOST:-prog-edu-assistant.salikh.info}

set -e

"$(dirname "$0")/docker/build.sh"
@execute docker tag server asia.gcr.io/prog-edu-assistant/server
@execute docker tag worker asia.gcr.io/prog-edu-assistant/worker
@execute docker push asia.gcr.io/prog-edu-assistant/server
@execute docker push asia.gcr.io/prog-edu-assistant/worker

@execute scp -r \
  deploy/{certs,docker-compose.yml,secret.env,service-account.json} \
  $GCE_HOST:

@execute ssh $GCE_HOST "mkdir -p logs && docker ps -q | xargs -n1 docker kill; cat service-account.json | docker login -u _json_key --password-stdin https://asia.gcr.io && docker pull asia.gcr.io/prog-edu-assistant/worker && docker pull asia.gcr.io/prog-edu-assistant/server && docker run -d --rm -v /var/run/docker.sock:/var/run/docker.sock -v \$PWD:\$PWD -w=\$PWD --entrypoint=sh docker/compose:1.24.0 -c 'cat service-account.json | docker login -u _json_key --password-stdin https://asia.gcr.io && docker-compose up --scale worker=2'"

echo OK
