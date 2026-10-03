// Package bootstrap é a "composition root": o único lugar que conhece todos
// os módulos e liga um no outro.
package bootstrap

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/httpx"
)

// NewApp monta a aplicação. db == nil usa repositórios em memória.
func NewApp(log *slog.Logger, db *pgxpool.Pool, problemTypeBase string) *fiber.App {
	app := httpx.NewApp(log, problemTypeBase)

	app.Get("/health", func(c fiber.Ctx) error { return c.SendString("ok") })

	usersMod := users.New(users.Deps{DB: db})
	ordersMod := orders.New(orders.Deps{DB: db, Users: usersMod.API()})

	v1 := app.Group("/v1")
	usersMod.RegisterRoutes(v1)
	ordersMod.RegisterRoutes(v1)

	return app
}
