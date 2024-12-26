# Description: Makefile for code-surgeon
# Author: Ovidiu Miron
#
#

.DEFAULT_GOAL := install



.PHONY: build install run-server new-chat continue-chat generate-proto ngrok test

install:
	go install ./cmd/code-surgeon

run-server:
	go run ./cmd/code-surgeon server

new-chat:
	go run ./cmd/code-surgeon chat

continue-chat:
	go run ./cmd/code-surgeon chat --chat-id=${CHAT_ID}

generate-proto:
	buf build
	buf generate
ngrok:
	ngrok http 8010

test:
	go build -o /dev/null ./cmd/code-surgeon/
	go build -o /dev/null ./chatcli/
	go build -o /dev/null ./cmd/code-surgeon/
	go test -v ./grpc/
	go test -v ./chatcli/

build-delve:
	go build -gcflags "all=-N -l" -o ./bin/code-surgeon ./cmd/code-surgeon

