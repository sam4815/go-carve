.PHONY: web
web:
	mkdir -p build
	cp $(shell go env GOROOT)/misc/wasm/wasm_exec.js build/
	cp -r web/* build/
	GOOS=js GOARCH=wasm go build -o build/main.wasm
