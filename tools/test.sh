#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list ./... 2>/dev/null | grep -v vendor | grep -v github.com/rackn/rocket-skates/client  | grep -v github.com/rackn/rocket-skates/models) ; do
  go test -race -coverprofile=profile.out -covermode=atomic $d
  if [ -f profile.out ]; then
    cat profile.out >> coverage.txt
    rm profile.out
  fi
done

