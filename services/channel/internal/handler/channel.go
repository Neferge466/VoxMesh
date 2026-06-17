package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/voxmesh/pkg/auth"
	voxerrors "github.com/voxmesh/pkg/errors"
	"github.com/voxmesh/pkg/model"

	chansvc "github.com/voxmesh/channel/internal/service"
)

type ChannelHandler struct {
	svc *chansvc.ChannelService
}

func NewChannelHandler(svc *chansvc.ChannelService) *ChannelHandler {
	return &ChannelHandler{svc: svc}
}

func (h *ChannelHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1/channels", auth.FiberAuthMiddleware)

	api.Get("/", h.List)
	api.Post("/", h.Create)
	api.Get("/:id", h.Get)
	api.Patch("/:id", h.Update)
	api.Delete("/:id", h.Delete)
	api.Post("/:id/join", h.Join)
	api.Post("/:id/leave", h.Leave)
	api.Get("/:id/members", h.Members)
	api.Post("/:id/kick/:uid", h.Kick)
	api.Post("/:id/move/:uid", h.Move)
}

func (h *ChannelHandler) List(c *fiber.Ctx) error {
	parentID := c.Query("parent_id")
	var pid *string
	if parentID != "" {
		pid = &parentID
	}
	channels, err := h.svc.GetChannels(c.Context(), pid)
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(channels)
}

func (h *ChannelHandler) Get(c *fiber.Ctx) error {
	ch, err := h.svc.GetChannel(c.Context(), c.Params("id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(ch)
}

func (h *ChannelHandler) Create(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	var req model.CreateChannelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request body"}})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "name required"}})
	}

	ch, err := h.svc.CreateChannel(c.Context(), req, claims.Subject)
	if err != nil {
		return handleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(ch)
}

func (h *ChannelHandler) Update(c *fiber.Ctx) error {
	var req model.UpdateChannelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request body"}})
	}

	ch, err := h.svc.UpdateChannel(c.Context(), c.Params("id"), req)
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(ch)
}

func (h *ChannelHandler) Delete(c *fiber.Ctx) error {
	if err := h.svc.DeleteChannel(c.Context(), c.Params("id")); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"message": "channel deleted"})
}

func (h *ChannelHandler) Join(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	var req model.JoinChannelRequest
	c.BodyParser(&req)

	if err := h.svc.JoinChannel(c.Context(), c.Params("id"), claims.Subject, "web", nil, req.Password); err != nil {
		log.Printf("[channel] join failed: channel=%s user=%s err=%v", c.Params("id"), claims.Subject, err)
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"message": "joined"})
}

func (h *ChannelHandler) Leave(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	if err := h.svc.LeaveChannel(c.Context(), c.Params("id"), claims.Subject); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"message": "left"})
}

func (h *ChannelHandler) Members(c *fiber.Ctx) error {
	members, err := h.svc.GetMembers(c.Context(), c.Params("id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(members)
}

func (h *ChannelHandler) Kick(c *fiber.Ctx) error {
	if err := h.svc.KickUser(c.Context(), c.Params("id"), c.Params("uid")); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"message": "user kicked"})
}

func (h *ChannelHandler) Move(c *fiber.Ctx) error {
	var req struct {
		ToChannelID string `json:"to_channel_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": 40000, "message": "invalid request body"}})
	}

	if err := h.svc.MoveUser(c.Context(), c.Params("id"), req.ToChannelID, c.Params("uid")); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"message": "user moved"})
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
