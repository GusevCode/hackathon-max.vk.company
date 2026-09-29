.PHONY: fmt test lint check build swagger docker-up docker-down

BACKEND_DIR := src/backend

fmt:
	go -C $(BACKEND_DIR) fmt ./...

test:
	go -C $(BACKEND_DIR) test -race -coverprofile=coverage.out ./...

lint:
	cd $(BACKEND_DIR) && golangci-lint run --config ../../.golangci.yml ./...

check: fmt test lint

build:
	go -C $(BACKEND_DIR) build ./cmd/control-bot

swagger:
	go -C $(BACKEND_DIR) run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g internal/infrastructure/httpserver/api.go -d . -o internal/infrastructure/httpserver/swagger --parseInternal --packageName swaggerdocs

docker-up:
	docker compose up --build bot

docker-down:
	docker compose down
