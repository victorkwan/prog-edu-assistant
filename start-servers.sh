#!/bin/bash

cd "$(dirname "$0")"
source ../venv/bin/activate

set -ve

# Start message queue:
#sudo /etc/init.d/rabbitmq-server start

# Start Jupyter notebook server
jupyter notebook &

cd go
mkdir -p ../tmp-uploads
# Start the autograder worker
go run cmd/worker/worker.go --autograder_dir=../tmp-autograder --logtostderr --v=3 &

# Start the upload server
go run cmd/uploadserver/main.go --logtostderr --v=3 --upload_dir=../tmp-uploads
