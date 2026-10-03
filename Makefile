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
