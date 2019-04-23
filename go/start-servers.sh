#!/bin/bash

cd "$(dirname "$0")"

set -ve
sudo /etc/init.d/rabbitmq-server start

# Generate student notebook
mkdir -p tmp-student
go run cmd/assign/assign.go --command student --input=../exercises/helloworld-en-master.ipynb --output=tmp-student/helloworld-en-student.ipynb

mkdir -p tmp-autograder
go run cmd/assign/assign.go --command=autograder --input=../exercises/helloworld-en-master.ipynb --output=tmp-autograder

. ../../venv/bin/activate
jupyter notebook &

mkdir -p uploads
go run cmd/worker/worker.go --autograder_dir=tmp-autograder --logtostderr --v=3 &
go run cmd/uploadserver/main.go --logtostderr --v=3 --upload_dir=uploads
