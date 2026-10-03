// Package httpapi expõe os casos de uso de usuários via REST (Fiber).
//
// Repare que nenhum handler escolhe status de erro: eles só fazem `return err`
// e o httpx.ErrorHandler monta o Problem Details. Handler fino = sem "gateway"
// engolindo erro e devolvendo 500 genérico.
package httpapi

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/app"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/modules/users/internal/domain"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/httpx"
)

type Handler struct {
	svc *app.Service
}

func New(svc *app.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/users")
	g.Post("/", h.create)
	g.Get("/", h.list)
	g.Get("/:id", h.get)
}

// Os DTOs HTTP são separados do domínio: mudar o JSON não mexe na regra de
// negócio, e vice-versa.
type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func toResponse(u domain.User) userResponse {
	return userResponse{ID: u.ID.String(), Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt}
}

func (h *Handler) create(c fiber.Ctx) error {
	var req createUserRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		return err
	}
	u, err := h.svc.Register(c.Context(), req.Name, req.Email)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(toResponse(u))
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
