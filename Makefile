# Description: Makefile for code-surgeon
# Author: Ovidiu Miron
#
#

.DEFAULT_GOAL := install



.PHONY: build install run-server generate-proto ngrok test

install:
	go install ./cmd/code-surgeon

run-server:
	go run ./cmd/code-surgeon server


generate-proto:
	buf build
	buf generate
ngrok:
	ngrok http 8010

test:
	go build -o /dev/null ./cmd/code-surgeon/
	go test -v ./...

build-delve:
	go build -gcflags "all=-N -l" -o ./bin/code-surgeon ./cmd/code-surgeon

