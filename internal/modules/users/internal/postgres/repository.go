// Package postgres implementa app.Repository com pgx.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/domain"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/apperr"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, u domain.User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users_users (id, name, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email, updated_at = EXCLUDED.updated_at`,
		u.ID, u.Name, u.Email, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, name, email, created_at, updated_at FROM users_users WHERE id = $1`, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	// Aqui o erro do driver vira erro de DOMÍNIO. Acima desta camada ninguém
	// sabe que existe Postgres.
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, apperr.Internal(err)
	}
	return u, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, email, created_at, updated_at FROM users_users ORDER BY created_at`)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.User, error) {
		var u domain.User
		err := row.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)
		return u, err //nolint:wrapcheck // embrulhado por apperr.Internal logo abaixo
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return users, nil
}
