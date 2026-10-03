# go-modular-monolith

Template de **monolith modular** em Go com Fiber v3, PostgreSQL (pgx) e erros
padronizados em **Problem Details (RFC 9457)**.

## Começando um projeto novo a partir deste template

```bash
go install golang.org/x/tools/cmd/gonew@latest
gonew github.com/dev-danilocordeiro/go-modular-monolith github.com/voce/meu-app
```

O `gonew` reescreve o module path em todos os imports. O botão
"Use this template" do GitHub também funciona, mas mantém o module path antigo.

## Rodando

```bash
cp .env.example .env
make up          # Postgres via docker compose
make run         # API em :8080 (migrations rodam no start)
make run-memory  # sem banco nenhum
make test
```

## Docker

```bash
make docker-build   # build da imagem e mostra o tamanho
make docker-up      # Postgres + API em containers (http://localhost:8080)
make docker-down
```

A imagem final tem **~21 MB** (o CI falha se passar de 40 MB):

| Decisão | Por quê |
|---|---|
| Multi-stage build | O toolchain do Go (~300 MB) fica só no estágio de build |
| `CGO_ENABLED=0` | Binário estático, sem depender de libc na imagem |
| `-trimpath -ldflags="-s -w"` | Remove a tabela de símbolos e a informação de debug (~30% menor) |
| `distroless/static:nonroot` | ~6 MB, com CA certs, tzdata e usuário não-root, sem shell (menos superfície de ataque) |
| `go.mod`/`go.sum` copiados antes do código | A camada de dependências fica em cache entre builds |
| `--mount=type=cache` | Cache de módulos e de compilação do BuildKit |
| `/api healthcheck` | Distroless não tem curl: o próprio binário faz o `HEALTHCHECK` |

Para ir abaixo de ~15 MB dá para usar `FROM scratch`, copiando manualmente os
CA certs e o `/etc/passwd`. Na prática, o ganho de ~6 MB raramente compensa.

## Estrutura

```
cmd/api/                    main: config, banco, graceful shutdown
internal/bootstrap/         composition root: liga os módulos
internal/platform/
  apperr/                   o erro de aplicação (Kind + Code), sem HTTP
  httpx/                    Fiber + tradução erro -> Problem Details
  config/ database/
internal/modules/<modulo>/
  <modulo>.go               API PÚBLICA: interface, DTOs, erros
  module.go                 monta o módulo (New, RegisterRoutes, API)
  internal/                 privado: o compilador impede import de fora
    domain/                 entidades, regras, erros do domínio
    app/                    casos de uso + portas (interfaces de repositório)
    postgres/ memory/       implementações do repositório
    httpapi/                handlers Fiber + DTOs HTTP
migrations/                 SQL embutido no binário (goose)
```

### Regras de fronteira

1. Um módulo só importa **o pacote raiz** de outro (`internal/modules/users`).
   Tentar importar `users/internal/...` de fora é **erro de compilação**
   (regra `internal/` do Go), sem precisar de linter.
2. Tabelas têm o prefixo do módulo (`users_users`, `orders_orders`) e não há
   foreign key entre módulos.
3. A interface que um módulo consome é declarada **no consumidor**
   (`orders/internal/app.Users`), com só os métodos usados.

## Erros: um único modelo para REST e chamadas internas

```
domain  ──► var ErrNotFound = apperr.New(KindNotFound, "users.not_found", "...")
repo    ──► pgx.ErrNoRows        → domain.ErrNotFound
            unique violation 23505 → domain.ErrEmailTaken.Wrap(err)
            outro erro do driver → apperr.Internal(err)
app     ──► só devolve (ou traduz) erros
handler ──► `return err` (nenhum handler escolhe status)
httpx   ──► ErrorHandler: Kind → status, monta application/problem+json
```

Outro módulo que chama `users.API.GetUser` recebe **o mesmo** `*apperr.Error`
que a rota `GET /v1/users/:id` transformaria em 404:

```go
u, err := usersAPI.GetUser(ctx, id)
if errors.Is(err, users.ErrNotFound) { ... }
```

O consumidor escolhe entre:

- **propagar** (`return err`) quando o significado não muda;
- **traduzir** quando muda: em `orders`, "usuário não existe" vira
  `orders.unknown_user` (422) com `ErrUnknownUser.Wrap(err)`. A cadeia é
  preservada, então `errors.Is(err, users.ErrNotFound)` continua `true`.

Resposta HTTP:

```json
{
  "type": "https://errors.example.com/users.invalid",
  "title": "Bad Request",
  "status": 400,
  "detail": "dados de usuário inválidos",
  "instance": "/v1/users",
  "code": "users.invalid",
  "errors": [
    { "field": "name",  "message": "obrigatório" },
    { "field": "email", "message": "e-mail inválido" }
  ]
}
```

Erros internos (banco fora, panic) viram `500` com `code: "internal"` e **sem
detalhes**: a causa real só aparece no log.

| Kind              | HTTP |
|-------------------|------|
| `KindInvalid`       | 400 |
| `KindUnauthorized`  | 401 |
| `KindForbidden`     | 403 |
| `KindNotFound`      | 404 |
| `KindConflict`      | 409 |
| `KindUnprocessable` | 422 |
| `KindInternal`      | 500 |

## Trocando de banco

Cada módulo depende só da interface `app.Repository`, que documenta o
**contrato de erros** (ex.: `GetByID` devolve `domain.ErrNotFound`). Para
outro banco, escreva uma nova implementação (ex.: `internal/mysql/`) que
respeite o contrato e troque em `module.go`. Domínio, casos de uso e handlers
não mudam.

## Adicionando um módulo

1. Copie `internal/modules/orders` como ponto de partida.
2. Declare os erros em `internal/domain` e reexporte no pacote raiz.
3. Crie a migration `migrations/0000N_<modulo>.sql` com tabelas prefixadas.
4. Registre o módulo em `internal/bootstrap/bootstrap.go`.
