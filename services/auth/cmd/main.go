package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/voxmesh/pkg/auth"
	"github.com/voxmesh/pkg/config"
	"github.com/voxmesh/pkg/db"
	slogx "github.com/voxmesh/pkg/log"
	"github.com/voxmesh/pkg/ratelimit"

	"github.com/voxmesh/auth/internal/handler"
	"github.com/voxmesh/auth/internal/service"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// Database
	pool, err := db.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slogx.Fatal("postgres: %v", err)
	}
	defer pool.Close()

	// Redis
	redisClient, err := db.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		slogx.Fatal("redis: %v", err)
	}
	defer redisClient.Close()

	// JWT keys
	if err := auth.LoadKeys(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath); err != nil {
		slogx.Fatal("jwt keys: %v", err)
	}

	// Service layer
	authSvc := service.NewAuthService(pool)

	// HTTP server
	app := fiber.New(fiber.Config{
		AppName:      "voxmesh-auth",
		ErrorHandler: defaultErrorHandler,
	})

	// Prometheus metrics
	fiberprometheus.New("voxmesh-auth").RegisterAt(app, "/metrics")

	// Rate limiting — strict on auth endpoints, loose on everything else
	rlStore := ratelimit.NewRedisStore(redisClient)
	app.Use(ratelimit.New(ratelimit.Config{
		Store:  rlStore,
		Max:    5,
		Window: time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			// Per-IP rate limit for login/register
			return "auth:" + c.IP() + ":" + c.Path()
		},
		SkipFunc: func(c *fiber.Ctx) bool {
			p := c.Path()
			return p != "/api/v1/auth/login" && p != "/api/v1/auth/register" && p != "/api/v1/auth/refresh"
		},
	}))

	// CORS — allow frontend dev server, localhost, and any extra via CORS_ORIGINS env var
	corsOriginsAuth := "http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000"
	if extra := os.Getenv("CORS_ORIGINS"); extra != "" {
		corsOriginsAuth = corsOriginsAuth + "," + extra
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOriginsAuth,
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		MaxAge:       int((10 * time.Minute).Seconds()),
	}))

	handler.NewAuthHandler(authSvc).RegisterRoutes(app)

	// Health check — verifies DB and Redis connectivity
	app.Get("/health", func(c *fiber.Ctx) error {
		status := fiber.Map{"status": "ok"}
		if err := pool.Ping(c.Context()); err != nil {
			status = fiber.Map{"status": "degraded", "postgres": "unreachable"}
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}
		if err := redisClient.Ping(c.Context()).Err(); err != nil {
			status = fiber.Map{"status": "degraded", "redis": "unreachable"}
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}
		return c.JSON(status)
	})

	slogx.Info("auth starting on %s", cfg.Address())

	go func() {
		if err := app.Listen(cfg.Address()); err != nil {
			slogx.Fatal("server listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slogx.Info("auth shutting down")
	app.Shutdown()
}

func defaultErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{
		"error": fiber.Map{"code": 50001, "message": err.Error()},
	})
}
