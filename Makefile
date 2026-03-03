.PHONY: build run test lint docker-up docker-down

build:
	go build -o server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
