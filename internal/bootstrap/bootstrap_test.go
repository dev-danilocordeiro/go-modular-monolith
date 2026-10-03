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
	"github.com/google/uuid"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/bootstrap"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/httpx"
)

// Testes ponta a ponta com repositórios em memória: exercitam rota, handler,
// serviço, comunicação entre módulos e o ErrorHandler de Problem Details.

func newApp(t *testing.T) *fiber.App {
	t.Helper()
	return bootstrap.NewApp(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, "https://errors.example.com/")
}

// response é o que os testes precisam da resposta, com o body já lido e fechado.
type response struct {
	StatusCode int
	Header     http.Header
}

func do(t *testing.T, app *fiber.App, method, path, body string) (response, []byte) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response{StatusCode: resp.StatusCode, Header: resp.Header}, b
}

func problemOf(t *testing.T, resp response, body []byte) httpx.Problem {
	t.Helper()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("Content-Type = %q, quer application/problem+json (body: %s)", ct, body)
	}
	var p httpx.Problem
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("body não é Problem: %v (%s)", err, body)
	}
	return p
}

func TestProblemDetails(t *testing.T) {
	app := newApp(t)

	// cria um usuário válido para os cenários de conflito e pedido
	resp, body := do(t, app, http.MethodPost, "/v1/users", `{"name":"Ana","email":"ana@example.com"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("criar usuário: %d %s", resp.StatusCode, body)
	}
	var created struct{ ID string }
	_ = json.Unmarshal(body, &created)

	tests := []struct {
		name, method, path, body string
		status                   int
		code                     string
		fields                   int
	}{
		{"validação acumula campos", "POST", "/v1/users", `{"name":"","email":"x"}`, 400, "users.invalid", 2},
		{"json malformado", "POST", "/v1/users", `{`, 400, "request.invalid_body", 0},
		{"e-mail duplicado", "POST", "/v1/users", `{"name":"Outra","email":"ANA@example.com"}`, 409, "users.email_taken", 0},
		{"usuário inexistente", "GET", "/v1/users/" + uuid.NewString(), "", 404, "users.not_found", 0},
		{"id malformado", "GET", "/v1/users/abc", "", 400, "users.invalid", 1},
		{"pedido p/ usuário inexistente (erro vindo de outro módulo)", "POST", "/v1/orders",
			`{"user_id":"` + uuid.NewString() + `","total_cents":100}`, 422, "orders.unknown_user", 0},
		{"rota inexistente", "GET", "/v1/nada", "", 404, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := do(t, app, tt.method, tt.path, tt.body)
			p := problemOf(t, resp, body)
			if resp.StatusCode != tt.status || p.Status != tt.status {
				t.Fatalf("status http=%d problem=%d, quer %d (%s)", resp.StatusCode, p.Status, tt.status, body)
			}
			if p.Code != tt.code {
				t.Fatalf("code = %q, quer %q", p.Code, tt.code)
			}
			if len(p.Errors) != tt.fields {
				t.Fatalf("errors = %v, quer %d campos", p.Errors, tt.fields)
			}
			if tt.code != "" && p.Type != "https://errors.example.com/"+tt.code {
				t.Fatalf("type = %q", p.Type)
			}
		})
	}

	t.Run("pedido válido", func(t *testing.T) {
		resp, body := do(t, app, "POST", "/v1/orders", `{"user_id":"`+created.ID+`","total_cents":1990}`)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("%d %s", resp.StatusCode, body)
		}
	})
}

func TestPanicBecomesOpaque500(t *testing.T) {
	app := newApp(t)
	app.Get("/boom", func(fiber.Ctx) error { panic("segredo interno") })

	resp, body := do(t, app, "GET", "/boom", "")
	p := problemOf(t, resp, body)
	if p.Status != 500 || p.Code != "internal" || strings.Contains(string(body), "segredo") {
		t.Fatalf("500 deveria ser genérico e não vazar detalhes: %s", body)
	}
}
