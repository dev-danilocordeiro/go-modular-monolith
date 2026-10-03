// Package app contém os casos de uso de pedidos.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/domain"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users"
)

// Repository: mesmo contrato de erros do módulo users.
//   - GetByID: domain.ErrNotFound se não existe.
type Repository interface {
	Create(ctx context.Context, o domain.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Order, error)
}

// Users é o que orders precisa do módulo users. Em Go, a interface fica
// no CONSUMIDOR e pede só os métodos que ele usa. users.API satisfaz esta
// interface sem saber que ela existe, e nos testes basta um fake de 1 método.
type Users interface {
	GetUser(ctx context.Context, id string) (users.User, error)
}

type Service struct {
	repo  Repository
	users Users
	now   func() time.Time
}

func NewService(repo Repository, u Users) *Service {
	return &Service{repo: repo, users: u, now: time.Now}
}

func (s *Service) Place(ctx context.Context, userID string, totalCents int64) (domain.Order, error) {
	o, err := domain.NewOrder(userID, totalCents, s.now())
	if err != nil {
		return domain.Order{}, err
	}

	if _, err := s.users.GetUser(ctx, userID); err != nil {
		// TRADUZIR quando o significado muda: "usuário não encontrado" no
		// contexto de criar pedido é uma regra de negócio violada (422).
		// Wrap mantém a causa: errors.Is(err, users.ErrNotFound) continua true.
		if errors.Is(err, users.ErrNotFound) {
			return domain.Order{}, domain.ErrUnknownUser.Wrap(err)
		}
		// PROPAGAR quando o significado não muda (ex.: banco fora -> 500).
		return domain.Order{}, err
	}

	if err := s.repo.Create(ctx, o); err != nil {
		return domain.Order{}, err
	}
	return o, nil
}

func (s *Service) Get(ctx context.Context, rawID string) (domain.Order, error) {
	id, err := domain.ParseID(rawID)
	if err != nil {
		return domain.Order{}, err
	}
	return s.repo.GetByID(ctx, id)
}
