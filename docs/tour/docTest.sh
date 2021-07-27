#!/bin/bash

function run_doc_test {
  echo -n "Checking $1... "
  if ! go run validate.go "$1"; then
    echo "failed"
    exit 1
  fi
  echo "done"
}

export -f run_doc_test

# find all posts and validate their quoted code contents matches the test files
find . -name index.md -exec bash -c 'run_doc_test "$0"' {} \; || exit 1

function run_test {
  echo -n "Running $1... "
  OLD_CWD=$(pwd)
  cd "$(dirname "$1")" || exit 1
  if ! OUTPUT=$(go run "$(basename "$1")"); then
    echo "failed"
    echo "$OUTPUT"
    exit 1
  fi
  echo "done"
  cd "$OLD_CWD" || exit 1
}

export -f run_test

# find all go test sources and check they compile and run
find . -name main.go -exec bash -c 'run_test "$0"' {} \; || exit 1
