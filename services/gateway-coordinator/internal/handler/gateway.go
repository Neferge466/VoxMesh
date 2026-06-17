package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/voxmesh/pkg/auth"
	voxerrors "github.com/voxmesh/pkg/errors"
	"github.com/voxmesh/pkg/model"

	"github.com/voxmesh/gateway-coordinator/internal/service"
)

type GatewayHandler struct {
	svc *service.GatewayCoordinator
}

func NewGatewayHandler(svc *service.GatewayCoordinator) *GatewayHandler {
	return &GatewayHandler{svc: svc}
}

func (h *GatewayHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1/gateways", auth.FiberAuthMiddleware)

	api.Get("/", h.List)
	api.Get("/:id", h.Get)
	api.Post("/", h.Create)
	api.Delete("/:id", h.Delete)
	api.Post("/:id/command", h.SendCommand)
}

func (h *GatewayHandler) List(c *fiber.Ctx) error {
	gws, err := h.svc.GetGateways(c.Context())
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(gws)
}

func (h *GatewayHandler) Get(c *fiber.Ctx) error {
	gw, err := h.svc.GetGateway(c.Context(), c.Params("id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(gw)
}

func (h *GatewayHandler) Create(c *fiber.Ctx) error {
	var req model.CreateGatewayRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request"}})
	}
	gw, err := h.svc.CreateGateway(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(gw)
}

func (h *GatewayHandler) Delete(c *fiber.Ctx) error {
	if err := h.svc.DeleteGateway(c.Context(), c.Params("id")); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"message": "gateway deleted"})
}

func (h *GatewayHandler) SendCommand(c *fiber.Ctx) error {
	var req model.GatewayCommandRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request"}})
	}
	if err := h.svc.SendCommand(c.Context(), c.Params("id"), req.Command, req.Params); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"message": "command sent"})
}

func handleError(c *fiber.Ctx, err error) error {
	if apiErr, ok := err.(*voxerrors.APIError); ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": apiErr})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"code": 50001, "message": err.Error()}})
}
