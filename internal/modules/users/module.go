package users

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/app"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/httpapi"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/memory"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/postgres"
)

// Deps são as dependências que o módulo recebe de fora.
type Deps struct {
	DB *pgxpool.Pool // nil => repositório em memória
}

// Module monta o módulo inteiro. O main só conhece Module, nunca as camadas.
type Module struct {
	svc *app.Service
}

func New(deps Deps) *Module {
	var repo app.Repository = memory.New()
	if deps.DB != nil {
		repo = postgres.New(deps.DB)
	}
	return &Module{svc: app.NewService(repo)}
}

func (m *Module) RegisterRoutes(r fiber.Router) {
	httpapi.New(m.svc).Register(r)
}

// API devolve a implementação do contrato público.
func (m *Module) API() API { return api{svc: m.svc} }

// api adapta o Service interno ao contrato público. Os erros passam sem
// tradução: o chamador recebe exatamente o que a rota REST receberia.
type api struct {
	svc *app.Service
}

func (a api) GetUser(ctx context.Context, id string) (User, error) {
	u, err := a.svc.Get(ctx, id)
	if err != nil {
		return User{}, err
	}
	return User{ID: u.ID.String(), Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt}, nil
}
