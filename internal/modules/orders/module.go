package orders

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/app"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/httpapi"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/memory"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/postgres"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users"
)

type Deps struct {
	DB    *pgxpool.Pool // nil => repositório em memória
	Users users.API     // dependência de outro módulo: só pela API pública
}

type Module struct {
	svc *app.Service
}

func New(deps Deps) *Module {
	var repo app.Repository = memory.New()
	if deps.DB != nil {
		repo = postgres.New(deps.DB)
	}
	return &Module{svc: app.NewService(repo, deps.Users)}
}

func (m *Module) RegisterRoutes(r fiber.Router) {
	httpapi.New(m.svc).Register(r)
}
