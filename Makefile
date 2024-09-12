generate-proto:
	buf build
	buf generate
ngrok:
	ngrok http 8002
