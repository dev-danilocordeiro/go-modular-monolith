// Package orders é a API pública do módulo de pedidos.
package orders

import "github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/orders/internal/domain"

var (
	ErrNotFound    = domain.ErrNotFound
	ErrInvalid     = domain.ErrInvalid
	ErrUnknownUser = domain.ErrUnknownUser
)
