// Package httpapi expõe os casos de uso de usuários via REST (Fiber).
//
// Repare que nenhum handler escolhe status de erro nem checa permissão: eles
// só fazem `return err`. A autorização está no Service e o
// httpx.ErrorHandler monta o Problem Details.
package httpapi

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/app"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/domain"
)

type Handler struct {
	svc *app.Service
}

func New(svc *app.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/users")
	g.Get("/me", h.me) // antes de /:id, senão "me" seria lido como um ID
	g.Get("/", h.list)
	g.Get("/:id", h.get)
}

// Os DTOs HTTP são separados do domínio: mudar o JSON não mexe na regra de
// negócio, e vice-versa.
type userResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func toResponse(u domain.User) userResponse {
	return userResponse{ID: u.ID, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt}
}

func (h *Handler) me(c fiber.Ctx) error {
	u, err := h.svc.Me(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(toResponse(u))
}

func (h *Handler) get(c fiber.Ctx) error {
	u, err := h.svc.Get(c.Context(), c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(toResponse(u))
}

func (h *Handler) list(c fiber.Ctx) error {
	users, err := h.svc.List(c.Context())
	if err != nil {
		return err
	}
	out := make([]userResponse, len(users))
	for i, u := range users {
		out[i] = toResponse(u)
	}
	return c.JSON(out)
}
