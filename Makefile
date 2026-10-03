.PHONY: run run-memory test lint up down tidy

-include .env
export

run: ## roda a API com Postgres (suba com `make up` antes)
	go run ./cmd/api

run-memory: ## roda a API sem banco
	DB_DRIVER=memory go run ./cmd/api

test:
	go test -race ./...

lint:
	go vet ./...
	golangci-lint run

up:
	docker compose up -d --wait

down:
	docker compose down

tidy:
	go mod tidy

IMAGE ?= go-modular-monolith
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: docker-build docker-up docker-down
docker-build: ## build da imagem e mostra o tamanho
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .
	@docker image ls $(IMAGE):latest --format 'tamanho: {{.Size}}'

docker-up: ## Postgres + API em containers
	docker compose --profile app up -d --build --wait

docker-down:
	docker compose --profile app down
