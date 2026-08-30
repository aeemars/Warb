.PHONY: run build clean test

# Load .env if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

run:
	@echo "Starting Proactive Opportunity Engine..."
	go run ./cmd/server

build:
	@echo "Building..."
	@mkdir -p bin
	go build -o bin/server ./cmd/server
	@echo "Built: bin/server"

clean:
	rm -rf bin/ data/

test:
	go test ./...

tidy:
	go mod tidy
