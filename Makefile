# Copyright 2016 EF CTX. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

all: test

test: lint gotest

lint:
	go get github.com/alecthomas/gometalinter honnef.co/go/unused/cmd/unused
	gometalinter --install --vendored-linters
	go install ./...
	gometalinter -j 4 --enable=misspell --enable=gofmt --enable=unused --disable=dupl --disable=errcheck --disable=gas --disable=interfacer --disable=gocyclo --deadline=10m --tests --vendor ./...

gotest:
	go test

coverage:
	go test -coverprofile=coverage.txt -covermode=atomic

prepare-test-server:
	@ test -n "$${TSURU_TEST_HOST}" || (echo >&2 "please define TSURU_TEST_HOST" && exit 3)
	@ test -n "$${TSURU_TEST_TOKEN}" || (echo >&2 "please define TSURU_TEST_TOKEN" && exit 3)
	TSURU_TARGET="$${TSURU_TEST_HOST}" TSURU_TOKEN="$${TSURU_TEST_TOKEN}" tsuru-admin platform-add python || tsuru-admin platform-update python
	TSURU_TARGET="$${TSURU_TEST_HOST}" TSURU_TOKEN="$${TSURU_TEST_TOKEN}" tsuru-admin pool-add -p 'dev\dev.example.com' || true
	TSURU_TARGET="$${TSURU_TEST_HOST}" TSURU_TOKEN="$${TSURU_TEST_TOKEN}" tsuru-admin pool-add -p 'qa\qa.example.com' || true
	TSURU_TARGET="$${TSURU_TEST_HOST}" TSURU_TOKEN="$${TSURU_TEST_TOKEN}" tsuru-admin pool-add -p 'stage\stage.example.com' || true
	TSURU_TARGET="$${TSURU_TEST_HOST}" TSURU_TOKEN="$${TSURU_TEST_TOKEN}" tsuru-admin pool-add -p 'prod\example.com' || true