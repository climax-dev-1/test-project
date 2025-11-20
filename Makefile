.PHONY: run generate test clean

# Run the server
run:
	go run server.go

# Generate GraphQL code
generate:
	go run github.com/99designs/gqlgen generate

# Download dependencies
deps:
	go mod download

# Run tests
test:
	go test -v ./...

# Clean generated files
clean:
	rm -rf graph/generated.go graph/model/models_gen.go

# Install gqlgen as a tool
install-tools:
	go install github.com/99designs/gqlgen@latest

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Build binary
build:
	go build -o bin/server server.go

