#!/bin/bash

set -ve

bash -ve ./build-student.sh

cd tmp/student

git init
git add . 
git commit -a -m 'Student notebooks'
git push -f git@github.com:salikh/student-notebooks.git
