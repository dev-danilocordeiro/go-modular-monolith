# AGENTS.md

Instruções para agentes de código (incluindo a linha de execução do Sentinela) que trabalham
neste repositório. Os comandos abaixo são os mesmos que o CI (`.github/workflows/ci.yml`) roda, e
todos passam na `main`.

## Setup

- Go na versão do `go.mod` (`go 1.27.0`; o CI usa `go-version-file: go.mod`).
- `golangci-lint` v2 (o CI usa a `latest`; config em `.golangci.yml`).
- Docker, só para subir Postgres e Keycloak ou para construir a imagem. Os testes não precisam.

## Build, testes e lint

```bash
go vet ./...
go test -race ./...      # = make test; não precisa de nada rodando (tokens são gerados nos testes)
golangci-lint run        # make lint roda go vet + golangci-lint
go build ./...
```

- O lint inclui `errorlint` (use `errors.Is`/`errors.As`), `wrapcheck` (embrulhe erros de libs
  externas), `bodyclose`, `noctx` e `revive`, e os formatadores `gofmt` e `goimports`.
- Use `go test -race -count=1 ./...` quando precisar que os testes rodem de novo sem cache.

## Imagem

```bash
docker build -t go-modular-monolith:dev .   # ou make docker-build
```

O CI falha se a imagem passar de **40 MB**. Não adicione nada ao estágio final (distroless).

## Rodando localmente

```bash
cp .env.example .env
make up          # Postgres + Keycloak via docker compose (portas 5432 e 8180)
make run         # API em :8080; as migrations rodam no start
make run-memory  # sem Postgres (o Keycloak continua necessário)
```

Se a porta 5432 já estiver em uso na máquina, o `postgres` do compose não sobe. Os testes não
dependem dele.

## Regras de fronteira entre módulos

Os módulos ficam em `internal/modules/<modulo>/`, e `internal/bootstrap/` liga todos eles.

1. Um módulo só importa **o pacote raiz** de outro (`internal/modules/users`), nunca
   `users/internal/...`. O próprio compilador impede.
2. As tabelas têm o prefixo do módulo (`users_users`, `orders_orders`), sem foreign key entre
   módulos.
3. A interface que um módulo consome é declarada **no consumidor** (ex.:
   `orders/internal/app.Users`), só com os métodos usados.
4. A autorização (RBAC/ABAC) fica em `internal/modules/<m>/internal/app/policies.go`. Toda
   operação nova chama `Check`.
5. Os erros seguem o modelo único de `internal/platform/apperr`: o mesmo `*apperr.Error` vale
   para REST e para chamadas entre módulos.

## Banco

- As migrations ficam em `migrations/` (`000NN_*.sql`, embutidas pelo `embed.go`). Só para
  frente: crie um arquivo novo e nunca edite uma migration que já está na `main`.

## Não mexa

A não ser que a issue peça explicitamente:

- `.github/` (CI), `Dockerfile`, `docker-compose.yml` e `deploy/keycloak/` (realm, clientes e
  usuários de dev);
- migrations que já estão na `main`;
- políticas de autorização de outro módulo além do que a issue pede;
- a versão do Go e as dependências no `go.mod`/`go.sum`: dependência nova é decisão de uma
  pessoa.
