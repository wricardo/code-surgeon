install:
	go install ./cmd/code-surgeon
generate-proto:
	buf build
	buf generate
ngrok:
	ngrok http 8002
