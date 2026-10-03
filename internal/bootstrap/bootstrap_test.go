package bootstrap_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/dev-danilocordeiro/go-authkit/authtest"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/bootstrap"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/httpx"
)

// Testes ponta a ponta com repositórios em memória e tokens assinados pelo
// authtest: exercitam rota, autenticação, políticas, comunicação entre
// módulos e o ErrorHandler de Problem Details, sem Keycloak nem banco.

type env struct {
	t   *testing.T
	app *fiber.App
	iss *authtest.Issuer
}

func newEnv(t *testing.T) *env {
	t.Helper()
	iss := authtest.NewIssuer(t)
	app := bootstrap.NewApp(bootstrap.Deps{
		Log:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		Verifier:        iss.Verifier(),
		ProblemTypeBase: "https://errors.example.com/",
	})
	return &env{t: t, app: app, iss: iss}
}

// response é o que os testes precisam da resposta, com o body já lido e fechado.
type response struct {
	Status int
	Header http.Header
	Body   []byte
}

func (e *env) do(token, method, path, body string) response {
	e.t.Helper()
	req := httptest.NewRequestWithContext(e.t.Context(), method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", authtest.Bearer(token))
	}
	resp, err := e.app.Test(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatal(err)
	}
	return response{Status: resp.StatusCode, Header: resp.Header, Body: b}
}

func (r response) problem(t *testing.T) httpx.Problem {
	t.Helper()
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("Content-Type = %q, quer application/problem+json (body: %s)", ct, r.Body)
	}
	var p httpx.Problem
	if err := json.Unmarshal(r.Body, &p); err != nil {
		t.Fatalf("body não é Problem: %v (%s)", err, r.Body)
	}
	return p
}

func (r response) id(t *testing.T) string {
	t.Helper()
	var v struct{ ID string }
	if err := json.Unmarshal(r.Body, &v); err != nil || v.ID == "" {
		t.Fatalf("resposta sem id: %s", r.Body)
	}
	return v.ID
}

func TestAuthAndPolicies(t *testing.T) {
	e := newEnv(t)
	var (
		alice   = e.iss.UserToken("alice", "orders:write")
		bob     = e.iss.UserToken("bob", "orders:write")
		admin   = e.iss.UserToken("root", "admin", "orders:write")
		noRoles = e.iss.UserToken("dave")
		billing = e.iss.ServiceToken("billing", "orders:write", "orders:read", "users:read")
		// sistema que pode criar pedidos, mas não pode ler usuários
		limited = e.iss.ServiceToken("limited", "orders:write")
	)

	// Pré-condições: alice e bob acessam /me (provisionamento just-in-time).
	for _, tok := range []string{alice, bob} {
		if r := e.do(tok, "GET", "/v1/users/me", ""); r.Status != 200 {
			t.Fatalf("/me: %d %s", r.Status, r.Body)
		}
	}
	aliceOrder := e.do(alice, "POST", "/v1/orders", `{"total_cents":1990}`)
	if aliceOrder.Status != http.StatusCreated {
		t.Fatalf("alice cria pedido: %d %s", aliceOrder.Status, aliceOrder.Body)
	}
	aliceOrderID := aliceOrder.id(t)

	tests := []struct {
		name, token, method, path, body string
		status                          int
		code                            string // vazio = sucesso, sem Problem
	}{
		// autenticação
		{"anônimo", "", "GET", "/v1/users/me", "", 401, "auth.unauthenticated"},
		{"token inválido", "lixo", "GET", "/v1/users/me", "", 401, "auth.invalid_token"},

		// users: RBAC + "só você mesmo"
		{"lê o próprio perfil por id", alice, "GET", "/v1/users/alice", "", 200, ""},
		{"perfil criado no 1º acesso por id", e.iss.UserToken("carol"), "GET", "/v1/users/carol", "", 200, ""},
		{"lê perfil de outro", alice, "GET", "/v1/users/bob", "", 403, "auth.forbidden"},
		{"admin lê perfil de outro", admin, "GET", "/v1/users/bob", "", 200, ""},
		{"serviço com users:read", billing, "GET", "/v1/users/bob", "", 200, ""},
		{"serviço sem users:read", limited, "GET", "/v1/users/bob", "", 403, "auth.forbidden"},
		{"serviço não tem /me", billing, "GET", "/v1/users/me", "", 403, "auth.forbidden"},
		{"lista sem ser admin", alice, "GET", "/v1/users", "", 403, "auth.forbidden"},
		{"admin lista", admin, "GET", "/v1/users", "", 200, ""},
		{"admin: usuário inexistente", admin, "GET", "/v1/users/ninguem", "", 404, "users.not_found"},

		// orders: RBAC (orders:write) + ABAC (dono do pedido)
		{"cria sem orders:write", noRoles, "POST", "/v1/orders", `{"total_cents":100}`, 403, "auth.forbidden"},
		{"cria pedido para outro", alice, "POST", "/v1/orders", `{"user_id":"bob","total_cents":100}`, 403, "auth.forbidden"},
		{"validação", alice, "POST", "/v1/orders", `{"total_cents":0}`, 400, "orders.invalid"},
		{"dono lê o pedido", alice, "GET", "/v1/orders/" + aliceOrderID, "", 200, ""},
		{"outro usuário lê o pedido", bob, "GET", "/v1/orders/" + aliceOrderID, "", 403, "auth.forbidden"},
		{"admin lê o pedido", admin, "GET", "/v1/orders/" + aliceOrderID, "", 200, ""},
		{"serviço com orders:read lê", billing, "GET", "/v1/orders/" + aliceOrderID, "", 200, ""},
		{"anônimo não descobre se o pedido existe", "", "GET", "/v1/orders/" + aliceOrderID, "", 401, "auth.unauthenticated"},

		// serviço criando em nome de alguém: a chamada interna orders -> users
		// passa pelas políticas de users com o MESMO principal
		{"serviço cria para alice", billing, "POST", "/v1/orders", `{"user_id":"alice","total_cents":500}`, 201, ""},
		{"serviço precisa informar user_id", billing, "POST", "/v1/orders", `{"total_cents":500}`, 400, "orders.invalid"},
		{"serviço: usuário inexistente vira 422", billing, "POST", "/v1/orders", `{"user_id":"ninguem","total_cents":500}`, 422, "orders.unknown_user"},
		{"serviço sem users:read barrado pelo módulo users", limited, "POST", "/v1/orders", `{"user_id":"alice","total_cents":500}`, 403, "auth.forbidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.do(tt.token, tt.method, tt.path, tt.body)
			if r.Status != tt.status {
				t.Fatalf("status = %d, quer %d (%s)", r.Status, tt.status, r.Body)
			}
			if tt.code == "" {
				return
			}
			p := r.problem(t)
			if p.Code != tt.code || p.Status != tt.status || p.Type != "https://errors.example.com/"+tt.code {
				t.Fatalf("problem = %+v, quer code %q", p, tt.code)
			}
			if tt.status == 401 {
				want := `Bearer realm="api"`
				if tt.code == "auth.invalid_token" {
					want += `, error="invalid_token"`
				}
				if got := r.Header.Get("WWW-Authenticate"); got != want {
					t.Fatalf("WWW-Authenticate = %q, quer %q", got, want)
				}
			}
		})
	}
}

func TestProblemDetailsBasics(t *testing.T) {
	e := newEnv(t)
	alice := e.iss.UserToken("alice", "orders:write")

	r := e.do(alice, "POST", "/v1/orders", `{`)
	if p := r.problem(t); p.Status != 400 || p.Code != "request.invalid_body" {
		t.Fatalf("json malformado: %+v", p)
	}
	r = e.do(alice, "POST", "/v1/orders", `{"total_cents":-1,"user_id":"  "}`)
	if p := r.problem(t); p.Status != 400 || len(p.Errors) != 1 {
		t.Fatalf("validação: %+v", p)
	}
	r = e.do("", "GET", "/rota/inexistente", "")
	if p := r.problem(t); p.Status != 404 {
		t.Fatalf("rota inexistente: %+v", p)
	}
}

func TestPanicBecomesOpaque500(t *testing.T) {
	e := newEnv(t)
	e.app.Get("/boom", func(fiber.Ctx) error { panic("segredo interno") })

	r := e.do("", "GET", "/boom", "")
	p := r.problem(t)
	if p.Status != 500 || p.Code != "internal" || strings.Contains(string(r.Body), "segredo") {
		t.Fatalf("500 deveria ser genérico e não vazar detalhes: %s", r.Body)
	}
}
