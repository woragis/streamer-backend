.PHONY: run run-worker build build-worker test tidy

run:
	go run ./cmd/server

run-worker:
	go run ./cmd/worker

build:
	go build -o bin/server ./cmd/server
	go build -o bin/worker ./cmd/worker

build-worker:
	go build -o bin/worker ./cmd/worker

test:
	go test ./...

tidy:
	go mod tidy
