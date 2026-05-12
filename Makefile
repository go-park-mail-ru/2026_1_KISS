.PHONY: build run test lint ci docker-up docker-down migrate docs fmt vet cover system-up generate proto proto-tools build-runner-image run-gateway run-auth run-notebook run-runner run-storage run-issue run-notification run-payment

build:
	go build -o gateway ./cmd/gateway
	go build -o auth ./cmd/auth
	go build -o notebook ./cmd/notebook
	go build -o runner ./cmd/runner
	go build -o storage ./cmd/storage
	go build -o payment ./cmd/payment

run: run-gateway

test:
	go test -race -coverprofile=coverage.out $$(go list ./... | grep -vE '(cmd/|internal/mocks|pkg/api/|/app$$|/grpc$$|internal/runner/container)')
	@go tool cover -func=coverage.out
	@TOTAL=$$(go tool cover -func=coverage.out | grep '^total:' | awk '{print $$3}' | tr -d '%'); \
	echo "Total coverage: $${TOTAL}%"; \
	awk "BEGIN { if ($${TOTAL}+0 < 70.0) { print \"FAIL: coverage $${TOTAL}% is below 70% threshold\"; exit 1 } }"

lint:
	golangci-lint run ./...

ci: proto generate lint test

build-runner-image:
	docker build -t kiss-python-runner -f build/py-runner/Dockerfile build/

docker-up: build-runner-image
	docker-compose up -d --build

docker-down:
	docker-compose down

migrate:
	go run ./cmd/migrator

docs:
	cd docs && npx redoc-cli build swagger.json --output index.html

generate:
	go generate ./...

run-gateway:
	go run ./cmd/gateway

run-auth:
	go run ./cmd/auth

run-notebook:
	go run ./cmd/notebook

run-runner:
	go run ./cmd/runner

run-storage:
	go run ./cmd/storage

run-issue:
	go run ./cmd/issue

run-notification:
	go run ./cmd/notification

run-payment:
	go run ./cmd/payment

proto-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto: proto-tools
	protoc --go_out=. --go_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		--go-grpc_out=. --go-grpc_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		api/proto/auth/auth.proto
	protoc --go_out=. --go_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		--go-grpc_out=. --go-grpc_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		api/proto/notebook/notebook.proto
	protoc --go_out=. --go_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		--go-grpc_out=. --go-grpc_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		api/proto/runner/runner.proto
	protoc --go_out=. --go_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		--go-grpc_out=. --go-grpc_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		api/proto/storage/storage.proto
	protoc --go_out=. --go_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
           --go-grpc_out=. --go-grpc_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
           api/proto/issue/issue.proto
	protoc --go_out=. --go_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		--go-grpc_out=. --go-grpc_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		api/proto/notification/notification.proto
	protoc --go_out=. --go_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		--go-grpc_out=. --go-grpc_opt=module=github.com/go-park-mail-ru/2026_1_KISS \
		api/proto/payment/payment.proto

fmt:
	go fmt ./...

vet:
	go vet ./...

cover:
	go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

system-up:
	@MISSING=0; \
	if command -v go > /dev/null 2>&1; then \
		printf "  [OK]     %-16s %s\n" "go" "$$(go version)"; \
	else \
		printf "  [MISS]   %-16s %s\n" "go" "https://go.dev/dl/"; \
		MISSING=$$((MISSING+1)); \
	fi; \
	if command -v docker > /dev/null 2>&1; then \
		printf "  [OK]     %-16s %s\n" "docker" "$$(docker --version)"; \
	else \
		printf "  [MISS]   %-16s %s\n" "docker" "https://docs.docker.com/get-docker/"; \
		MISSING=$$((MISSING+1)); \
	fi; \
	if command -v docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1; then \
		printf "  [OK]     %-16s %s\n" "docker compose" "$$(docker compose version)"; \
	elif command -v docker-compose > /dev/null 2>&1; then \
		printf "  [OK]     %-16s %s\n" "docker-compose" "$$(docker-compose --version)"; \
	else \
		printf "  [MISS]   %-16s %s\n" "docker compose" "https://docs.docker.com/compose/install/"; \
		MISSING=$$((MISSING+1)); \
	fi; \
	if command -v golangci-lint > /dev/null 2>&1; then \
		printf "  [OK]     %-16s %s\n" "golangci-lint" "$$(golangci-lint --version 2>&1 | awk '{print $$1, $$4}')"; \
	else \
		printf "  [MISS]   %-16s %s\n" "golangci-lint" "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		MISSING=$$((MISSING+1)); \
	fi; \
	if command -v npx > /dev/null 2>&1; then \
		printf "  [OK]     %-16s %s\n" "npx" "v$$(npx --version)"; \
	else \
		printf "  [MISS]   %-16s %s\n" "npx" "установи Node.js: https://nodejs.org/"; \
		MISSING=$$((MISSING+1)); \
	fi; \
	if command -v lefthook > /dev/null 2>&1; then \
		printf "  [OK]     %-16s %s\n" "lefthook (opt)" "v$$(lefthook version)"; \
	else \
		printf "  [--]     %-16s %s\n" "lefthook (opt)" "go install github.com/evilmartians/lefthook/v2@v2.1.2"; \
	fi; \
	echo ""; \
	[ $$MISSING -eq 0 ]
