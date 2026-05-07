.PHONY: run build test clean fmt vet

# Run the server in development mode.
run:
	go run ./cmd/server

# Build the production binary.
build:
	go build -ldflags="-s -w" -o bin/server.exe ./cmd/server

# Run all tests with verbose output and race detection.
test:
	go test -v -race ./...

# Format all Go files.
fmt:
	go fmt ./...

# Run Go vet for static analysis.
vet:
	go vet ./...

# Clean build artifacts.
clean:
	rm -rf bin/
