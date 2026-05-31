.PHONY: build generate_defaults test

generate_defaults:
	@echo "Generating default configuration from default-config.yml..."
	go run tools/generate_defaults.go

build: generate_defaults
	@echo "Building optimized binary..."
	go build -trimpath -buildmode=pie -gcflags="-l -B -C" -ldflags="-s -w -extldflags=-static" -o tgsm .
	@echo "Binary size:"
	@ls -lh tgsm
	@echo "Build complete!"

test: generate_defaults
	go test ./...
