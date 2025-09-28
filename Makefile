# Description: Makefile for code-surgeon
# Author: Ovidiu Miron
#
#

.DEFAULT_GOAL := install



.PHONY: build install test

install:
	go install ./cmd/code-surgeon

test:
	go build -o /dev/null ./cmd/code-surgeon/
	go test -v ./...

build-delve:
	go build -gcflags "all=-N -l" -o ./bin/code-surgeon ./cmd/code-surgeon

