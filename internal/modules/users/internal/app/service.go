// Package app contém os casos de uso de usuários. Orquestra domínio e
// persistência, mas depende só de interfaces ("portas").
package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/domain"
)

// Repository é a porta de persistência. Postgres e memória a implementam;
// trocar de banco = escrever outra implementação, sem mexer aqui.
//
// Contrato de erros (toda implementação DEVE respeitar):
//   - Create:  domain.ErrEmailTaken se o e-mail já existe.
//   - GetByID: domain.ErrNotFound se não existe.
//   - Falhas de infraestrutura: qualquer outro erro (vira 500 na borda).
type Repository interface {
	Create(ctx context.Context, u domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
}

type Service struct {
	repo Repository
	now  func() time.Time // injetável para testes determinísticos
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) Register(ctx context.Context, name, email string) (domain.User, error) {
	u, err := domain.NewUser(name, email, s.now())
	if err != nil {
		return domain.User{}, err
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

func (s *Service) Get(ctx context.Context, rawID string) (domain.User, error) {
	id, err := domain.ParseID(rawID)
	if err != nil {
		return domain.User{}, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}
