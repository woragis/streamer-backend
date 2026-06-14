.PHONY: run build test tidy docker-up docker-down

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

docker-up:
	docker compose up -d redis

docker-down:
	docker compose down
