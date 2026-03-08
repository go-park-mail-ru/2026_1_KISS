.PHONY: build run test lint ci docker-up docker-down migrate docx

build:
	go build -o server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

ci: lint test

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate:
	go run ./cmd/migrator

docs:
	cd docs && npx redoc-cli build swagger.json --output index.html
