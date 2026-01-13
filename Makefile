.PHONY: all test lint build clean example update-data

# Default target
all: lint test build

# Run tests
test:
	go test -v -race -cover ./...

# Run tests with coverage report
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	golangci-lint run

# Build the package
build:
	go build ./...

# Build example
example:
	go build -o bin/example ./example/...

# Run example
run-example:
	go run ./example/...

# Clean build artifacts
clean:
	rm -rf bin/ dist/ coverage.out coverage.html
	go clean

# Update embedded data from cloudip-db
update-data:
	curl -L -o data/cloudip.msgpack https://github.com/rezmoss/cloudip-db/raw/main/data/cloudip.msgpack
	@echo "Data updated. Don't forget to commit the changes."

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Check for outdated dependencies
deps:
	go list -u -m all

# Tidy dependencies
tidy:
	go mod tidy

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
