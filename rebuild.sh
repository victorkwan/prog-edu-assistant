#!/bin/bash

cd "$(dirname "$0")"

set -ve
rm -rf tmp-*
mkdir -p tmp-student tmp-autograder tmp-uploads

cd go
# Generate student notebook
go run cmd/assign/assign.go --command student --input=../exercises/helloworld-en-master.ipynb --output=../tmp-student/helloworld-en-student.ipynb

# Generate the autograder script directories
go run cmd/assign/assign.go --command=autograder --input=../exercises/helloworld-en-master.ipynb --output=../tmp-autograder
