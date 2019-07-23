#!/bin/bash
#
# A convenience script that rebuilds all student notebooks
# and prepares a directory to push to student github repo.

cd "$(dirname "$0")"
set -ve

rm -rf tmp/
bazel build ...
tar xvf bazel-genfiles/exercises/tmp-autograder_tar.tar
tar xvf bazel-genfiles/exercises/tmp-student_notebooks_tar.tar

cp -v student/* tmp/student/
cp -rv nbextensions tmp/student/

(cd tmp/student && git init && git add . && git commit -a -m 'Student notebooks')
