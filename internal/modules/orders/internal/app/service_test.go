package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/app"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/domain"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/memory"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/apperr"
)

// fakeUsers implementa app.Users: como a interface tem 1 método, o fake é trivial.
type fakeUsers struct{ err error }

func (f fakeUsers) GetUser(_ context.Context, id string) (users.User, error) {
	if f.err != nil {
		return users.User{}, f.err
	}
	return users.User{ID: id}, nil
}

func TestPlace(t *testing.T) {
	infraErr := apperr.Internal(errors.New("connection refused"))

	tests := []struct {
		name      string
		users     fakeUsers
		userID    string
		total     int64
		wantErr   error
		wantKind  apperr.Kind
		alsoMatch error // erro original que deve continuar na cadeia
	}{
		{name: "ok", userID: uuid.NewString(), total: 1000},
		{name: "validação", userID: "nope", total: 0, wantErr: domain.ErrInvalid, wantKind: apperr.KindInvalid},
		{
			name: "usuário inexistente é traduzido para 422", users: fakeUsers{err: users.ErrNotFound},
			userID: uuid.NewString(), total: 1000,
			wantErr: domain.ErrUnknownUser, wantKind: apperr.KindUnprocessable, alsoMatch: users.ErrNotFound,
		},
		{
			name: "falha de infra é propagada", users: fakeUsers{err: infraErr},
			userID: uuid.NewString(), total: 1000,
			wantErr: infraErr, wantKind: apperr.KindInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := app.NewService(memory.New(), tt.users)
			_, err := svc.Place(context.Background(), tt.userID, tt.total)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("erro inesperado: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, quer %v", err, tt.wantErr)
			}
			if got := apperr.KindOf(err); got != tt.wantKind {
				t.Fatalf("kind = %v, quer %v", got, tt.wantKind)
			}
			if tt.alsoMatch != nil && !errors.Is(err, tt.alsoMatch) {
				t.Fatalf("causa original %v perdida na cadeia", tt.alsoMatch)
			}
		})
	}
}
