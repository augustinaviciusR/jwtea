BINARY := jwtea
HOST ?= 127.0.0.1
PORT ?= 8080
CONFIG ?= dev.yaml

.PHONY: deps build run serve lint test tidy clean demo-flow

deps:
	go mod tidy

build:
	go build -o $(BINARY) main.go

# Default: run with config file and dashboard (always enabled)
run:
	go run main.go serve --config $(CONFIG) --host $(HOST) --port $(PORT)

# Alias for backwards compatibility
serve: run

lint:
	golangci-lint run --fix

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
