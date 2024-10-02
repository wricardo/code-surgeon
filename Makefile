install:
	go install ./cmd/code-surgeon
generate-proto:
	buf build
	buf generate
ngrok:
	ngrok http 8010

test:
	go build -o /dev/null ./chatcli/
	go build -o /dev/null ./cmd/code-surgeon/
