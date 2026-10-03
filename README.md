# go-modular-monolith

Template de **monolith modular** em Go com Fiber v3, PostgreSQL (pgx),
autenticação/autorização com **Keycloak** (RBAC + ABAC via
[go-authkit](https://github.com/dev-danilocordeiro/go-authkit)) e erros
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
make up          # Postgres + Keycloak via docker compose
make run         # API em :8080 (migrations rodam no start)
make run-memory  # sem Postgres (o Keycloak continua necessário)
make test        # não precisa de nada rodando: tokens são gerados nos testes
```

```bash
TOKEN=$(make -s token U=alice)          # alice | bob | admin (senha = usuário)
curl localhost:8080/v1/users/me -H "Authorization: Bearer $TOKEN"
curl localhost:8080/v1/orders -H "Authorization: Bearer $TOKEN" -d '{"total_cents":1990}' -H 'content-type: application/json'

TOKEN=$(make -s token-service)          # billing-service (client credentials)
```

Console do Keycloak: http://localhost:8180 (admin/admin).

## Docker

```bash
make docker-build   # build da imagem e mostra o tamanho
make docker-up      # Postgres + Keycloak + API em containers (http://localhost:8080)
make docker-down
```

A imagem final tem **~15 MB** (o CI falha se passar de 40 MB):

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
    app/                    casos de uso, políticas de acesso, portas (repositórios)
    postgres/ memory/       implementações do repositório
    httpapi/                handlers Fiber + DTOs HTTP
migrations/                 SQL embutido no binário (goose)
deploy/keycloak/            realm "app" importado pelo Keycloak de dev
```

### Regras de fronteira

1. Um módulo só importa **o pacote raiz** de outro (`internal/modules/users`).
   Tentar importar `users/internal/...` de fora é **erro de compilação**
   (regra `internal/` do Go), sem precisar de linter.
2. Tabelas têm o prefixo do módulo (`users_users`, `orders_orders`) e não há
   foreign key entre módulos.
3. A interface que um módulo consome é declarada **no consumidor**
   (`orders/internal/app.Users`), com só os métodos usados.

## Autenticação e autorização

Identidade é do **Keycloak**: login, senha, MFA e recuperação de conta ficam
lá. A API é só um *resource server*: valida o JWT (assinatura via JWKS, `iss`,
`aud`, `exp`, `typ`) e o transforma num `authkit.Principal`.

```
Pessoa  ── app-web: authorization code + PKCE ──► JWT ─┐
Sistema ── billing-service: client credentials ─► JWT ─┴─► fiberauth ─► Principal no ctx
                                                                         │
                                       políticas na camada app ◄─────────┘
```

Pessoa e sistema chegam pelo mesmo caminho. `Principal.Kind` diz qual é,
quando a regra precisar diferenciar.

### Onde a autorização acontece

**Na camada `app`, não no handler.** O middleware só autentica (token
inválido → 401; sem token → segue anônimo). Quem decide é a política do caso
de uso, e o principal viaja no `context.Context`. Por isso a **mesma regra
vale para chamadas internas**: quando `orders` chama `users.API.GetUser`, as
políticas de `users` são aplicadas ao mesmo principal. Um sistema que pode
criar pedidos, mas não tem `users:read`, recebe 403 vindo do módulo `users`.

### RBAC + ABAC

As políticas ficam em `internal/modules/<m>/internal/app/policies.go`. Um
arquivo por módulo responde "quem pode o quê":

```go
// RBAC (precisa de orders:write) E ABAC (para si, ou admin, ou sistema)
canPlaceFor = authz.All(
	authz.Role[string]("orders:write"),
	authz.Any(isSelf, authz.Role[string]("admin"), authz.Service[string]()),
)

// ABAC sobre o pedido carregado: dono, admin ou sistema com orders:read
canRead = authz.Any(isOwner, authz.Role[Order]("admin"),
	authz.All(authz.Service[Order](), authz.Role[Order]("orders:read")))
```

| Papel (client role de `app-api`) | Quem tem no realm de dev |
|---|---|
| `orders:write` | alice, bob, admin, billing-service |
| `admin` | admin |
| `users:read`, `orders:read` | billing-service |

Atributos próprios do token (ex.: `tenant_id`) ficam em `p.Attr("tenant_id")`
para regras ABAC mais ricas. Se as regras crescerem muito, a interface
`authz.Policy` permite trocar a implementação por OPA ou Cedar sem mexer nos
casos de uso.

### Usuários

O módulo `users` guarda só o **perfil local**, com `id` = `sub` do token. O
perfil é criado no primeiro acesso (`GET /v1/users/me`) e nome/e-mail são
sincronizados a cada acesso.

| Rota | Quem pode |
|---|---|
| `GET /v1/users/me` | qualquer pessoa autenticada |
| `GET /v1/users/:id` | a própria pessoa, `admin`, ou sistema com `users:read` |
| `GET /v1/users` | `admin` |
| `POST /v1/orders` | `orders:write`, para si (ou para qualquer um se `admin`/sistema) |
| `GET /v1/orders/:id` | o dono, `admin`, ou sistema com `orders:read` |

### Keycloak de dev × produção

O realm em `deploy/keycloak/realm-app.json` é **só para desenvolvimento**:
senhas iguais ao usuário, secret do `billing-service` fixo e *direct access
grants* ligados no `app-web` (é o que permite o `make token`). Em produção:
desligue *direct access grants*, use secrets gerados e HTTPS, e rode o
Keycloak em modo `start` com banco próprio.

Pontos que custam caro descobrir:

- **Audiência:** os clients que chamam a API precisam de um *audience mapper*
  que coloque `app-api` no `aud`. Sem isso, o token é rejeitado (e deve ser).
- **`client_id` em tokens de serviço:** vem do client scope
  `service_account`, que **não é vinculado automaticamente a clients
  importados**. O realm declara isso explicitamente no `billing-service`.
- **Emissor × rede do Docker:** o `iss` é `http://localhost:8180/...`, mas a
  API em container alcança o Keycloak por `http://keycloak:8080`. Por isso
  existe o `OIDC_DISCOVERY_URL` (e `KC_HOSTNAME_BACKCHANNEL_DYNAMIC=true`).

## Erros: um único modelo para REST e chamadas internas

```
domain  ──► var ErrNotFound = apperr.New(KindNotFound, "users.not_found", "...")
repo    ──► pgx.ErrNoRows        → domain.ErrNotFound
            outro erro do driver → apperr.Internal(err)
authz   ──► authkit.ErrUnauthenticated / ErrForbidden (convertidos por
            apperr.As via interfaces comportamentais, sem importar o authkit)
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
  "type": "https://errors.example.com/orders.invalid",
  "title": "Bad Request",
  "status": 400,
  "detail": "dados de pedido inválidos",
  "instance": "/v1/orders",
  "code": "orders.invalid",
  "errors": [
    { "field": "user_id",     "message": "obrigatório" },
    { "field": "total_cents", "message": "deve ser maior que zero" }
  ]
}
```

Um `401` também traz `WWW-Authenticate: Bearer realm="api"` (com
`error="invalid_token"` quando o token é inválido), como pede a RFC 6750.

Erros internos (banco fora, panic) viram `500` com `code: "internal"` e **sem
detalhes**: a causa real só aparece no log.

| Kind              | HTTP |
|-------------------|------|
| `KindInvalid`       | 400 |
| `KindUnauthorized`  | 401 (`auth.unauthenticated`, `auth.invalid_token`) |
| `KindForbidden`     | 403 (`auth.forbidden`) |
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
3. Escreva as políticas em `internal/app/policies.go` e chame `Check` em cada
   caso de uso. Novos papéis: crie como client role de `app-api` no Keycloak.
4. Crie a migration `migrations/0000N_<modulo>.sql` com tabelas prefixadas.
5. Registre o módulo em `internal/bootstrap/bootstrap.go`.
