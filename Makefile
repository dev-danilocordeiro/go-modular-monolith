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

# ---- Tokens de desenvolvimento (Keycloak do docker compose) ----
KC_TOKEN_URL ?= http://localhost:8180/realms/app/protocol/openid-connect/token
U ?= alice

.PHONY: token token-service
token: ## access token de pessoa: make token U=alice|bob|admin (senha = usuário)
	@curl -s $(KC_TOKEN_URL) -d grant_type=password -d client_id=app-web \
		-d username=$(U) -d password=$(U) | python3 -c 'import json,sys; print(json.load(sys.stdin)["access_token"])'

token-service: ## access token do billing-service (client credentials)
	@curl -s $(KC_TOKEN_URL) -d grant_type=client_credentials -d client_id=billing-service \
		-d client_secret=billing-secret | python3 -c 'import json,sys; print(json.load(sys.stdin)["access_token"])'
