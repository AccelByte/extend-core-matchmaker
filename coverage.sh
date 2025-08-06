#!/usr/bin/env bash

set -x
set -e
set -o pipefail

rm coverage.txt || true 2> /dev/null
# go install github.com/jstemmer/go-junit-report@latest

go test -tags=musl -v -count=1 -race -coverprofile=profile.out ./pkg/... | tee -a >(go-junit-report > test.xml) coverage.txt

# go install github.com/axw/gocov/gocov@latest
# go install github.com/AlekSi/gocov-xml@latest

gocov convert profile.out | gocov-xml > coverage.xml
# Workaround for bitbucket pipelines
sed -i '2d' coverage.xml
