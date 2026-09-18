.PHONY: fmt test lint check build docker-up docker-down

BACKEND_DIR := src/backend

fmt:
	go -C $(BACKEND_DIR) fmt ./...

test:
	go -C $(BACKEND_DIR) test -race -coverprofile=coverage.out ./...

lint:
	cd $(BACKEND_DIR) && golangci-lint run --config ../../.golangci.yml ./...

check: fmt test lint

build:
	go -C $(BACKEND_DIR) build ./cmd/echo-bot

docker-up:
	docker compose up --build bot

docker-down:
	docker compose down
