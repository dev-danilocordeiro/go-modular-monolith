// Package users é a API PÚBLICA do módulo de usuários: o único pacote dele
// que outros módulos podem importar. Tudo em users/internal/ é privado e o
// compilador do Go impede o import de fora.
package users

import (
	"context"
	"time"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/domain"
)

// User é o DTO público. Não é a entidade de domínio: o módulo pode mudar a
// entidade internamente sem quebrar quem consome.
type User struct {
	ID        string
	Name      string
	Email     string
	CreatedAt time.Time
}

// API é o contrato que outros módulos usam.
//
// Os erros devolvidos são os MESMOS *apperr.Error que a rota REST devolve
// (ex.: GetUser com ID inexistente -> ErrNotFound, que no HTTP vira 404 com
// code "users.not_found"). Compare com errors.Is(err, users.ErrNotFound).
type API interface {
	GetUser(ctx context.Context, id string) (User, error)
}

// Reexporta os erros do domínio para quem está fora do módulo.
var (
	ErrNotFound   = domain.ErrNotFound
	ErrEmailTaken = domain.ErrEmailTaken
	ErrInvalid    = domain.ErrInvalid
)
