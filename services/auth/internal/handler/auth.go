package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/voxmesh/pkg/auth"
	voxerrors "github.com/voxmesh/pkg/errors"
	"github.com/voxmesh/pkg/model"

	authsvc "github.com/voxmesh/auth/internal/service"
)

type AuthHandler struct {
	svc *authsvc.AuthService
}

func NewAuthHandler(svc *authsvc.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1/auth")
	api.Post("/register", h.Register)
	api.Post("/login", h.Login)
	api.Post("/refresh", h.Refresh)

	// Protected routes
	protected := api.Group("", auth.FiberAuthMiddleware)
	protected.Post("/logout", h.Logout)
	protected.Get("/me", h.Me)
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request body"}})
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "username, email, password required"}})
	}

	pair, err := h.svc.Register(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(pair)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request body"}})
	}

	pair, err := h.svc.Login(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(pair)
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request body"}})
	}

	pair, err := h.svc.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(pair)
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request body"}})
	}

	if err := h.svc.Logout(c.Context(), req.RefreshToken); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"message": "logged out"})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*auth.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": fiber.Map{"code": 40004, "message": "not authenticated"}})
	}

	user, err := h.svc.GetUser(c.Context(), claims.Subject)
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(user)
}

func handleError(c *fiber.Ctx, err error) error {
	if apiErr, ok := err.(*voxerrors.APIError); ok {
		status := fiber.StatusInternalServerError
		switch {
		case apiErr.Code >= 40000 && apiErr.Code < 41000:
			status = fiber.StatusUnauthorized
		case apiErr.Code >= 41000 && apiErr.Code < 43000:
			status = fiber.StatusBadRequest
		case apiErr.Code >= 43000 && apiErr.Code < 50000:
			status = fiber.StatusTooManyRequests
		case apiErr.Code >= 50000:
			status = fiber.StatusInternalServerError
		}
		return c.Status(status).JSON(fiber.Map{"error": apiErr})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"code": 50001, "message": err.Error()}})
}
