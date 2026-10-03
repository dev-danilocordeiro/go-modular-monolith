// Package bootstrap é a "composition root": o único lugar que conhece todos
// os módulos e liga um no outro.
package bootstrap

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-danilocordeiro/go-authkit"
	"github.com/dev-danilocordeiro/go-authkit/fiberauth"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/httpx"
)

type Deps struct {
	Log             *slog.Logger
	DB              *pgxpool.Pool    // nil => repositórios em memória
	Verifier        authkit.Verifier // valida os tokens (Keycloak em produção, authtest nos testes)
	ProblemTypeBase string
}

func NewApp(d Deps) *fiber.App {
	app := httpx.NewApp(d.Log, d.ProblemTypeBase)

	// Rotas públicas ficam ANTES do middleware de autenticação.
	app.Get("/health", func(c fiber.Ctx) error { return c.SendString("ok") })

	usersMod := users.New(users.Deps{DB: d.DB})
	ordersMod := orders.New(orders.Deps{DB: d.DB, Users: usersMod.API()})

	v1 := app.Group("/v1", fiberauth.New(d.Verifier))
	usersMod.RegisterRoutes(v1)
	ordersMod.RegisterRoutes(v1)

	return app
}
