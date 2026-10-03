// Package memory implementa app.Repository em memória. Serve para testes e
// para rodar a aplicação sem banco (DB_DRIVER=memory).
package memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/domain"
)

type Repository struct {
	mu    sync.RWMutex
	users map[uuid.UUID]domain.User
}

func New() *Repository {
	return &Repository{users: make(map[uuid.UUID]domain.User)}
}

func (r *Repository) Create(_ context.Context, u domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if strings.EqualFold(existing.Email, u.Email) {
			return domain.ErrEmailTaken
		}
	}
	r.users[u.ID] = u
	return nil
}

func (r *Repository) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (r *Repository) List(_ context.Context) ([]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.User, 0, len(r.users))
	for _, u := range r.users {
		out = append(out, u)
	}
	slices.SortFunc(out, func(a, b domain.User) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}
