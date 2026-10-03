// Package domain contém as regras de negócio de usuários. Não conhece HTTP,
// SQL nem Fiber, só Go puro.
package domain

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/apperr"
)

// Erros do módulo. São a "linguagem de erro" de users: o handler REST, o
// módulo orders e os testes comparam contra estas variáveis com errors.Is.
var (
	ErrNotFound   = apperr.New(apperr.KindNotFound, "users.not_found", "usuário não encontrado")
	ErrEmailTaken = apperr.New(apperr.KindConflict, "users.email_taken", "e-mail já cadastrado")
	ErrInvalid    = apperr.New(apperr.KindInvalid, "users.invalid", "dados de usuário inválidos")
)

type User struct {
	ID        uuid.UUID
	Name      string
	Email     string
	CreatedAt time.Time
}

// NewUser valida e cria um usuário. Acumula todos os erros de campo em vez
// de parar no primeiro, para o cliente corrigir tudo de uma vez.
func NewUser(name, email string, now time.Time) (User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	verr := ErrInvalid
	if name == "" {
		verr = verr.WithField("name", "obrigatório")
	} else if len(name) > 120 {
		verr = verr.WithField("name", "máximo de 120 caracteres")
	}
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		verr = verr.WithField("email", "e-mail inválido")
	}
	if verr.HasFields() {
		return User{}, verr
	}

	return User{ID: uuid.New(), Name: name, Email: email, CreatedAt: now.UTC()}, nil
}

// ParseID converte o ID textual. ID malformado é erro de entrada (400), não 404.
func ParseID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, ErrInvalid.WithField("id", "deve ser um UUID").Wrap(err)
	}
	return id, nil
}
