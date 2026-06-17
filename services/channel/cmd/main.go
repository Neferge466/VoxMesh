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

	"github.com/voxmesh/channel/internal/handler"
	"github.com/voxmesh/channel/internal/service"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slogx.Fatal("postgres: %v", err)
	}
	defer pool.Close()

	redisClient, err := db.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		slogx.Fatal("redis: %v", err)
	}
	defer redisClient.Close()

	_ = redisClient // used for temp channel TTL

	// JWT public key for token verification
	if err := auth.LoadKeys(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath); err != nil {
		slogx.Fatal("jwt keys: %v", err)
	}

	channelSvc := service.NewChannelService(pool)

	app := fiber.New(fiber.Config{AppName: "voxmesh-channel"})

	// Prometheus metrics
	fiberprometheus.New("voxmesh-channel").RegisterAt(app, "/metrics")

	corsOriginsCh := "http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000"
	if extra := os.Getenv("CORS_ORIGINS"); extra != "" {
		corsOriginsCh = corsOriginsCh + "," + extra
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOriginsCh,
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		MaxAge:       int((10 * time.Minute).Seconds()),
	}))
	handler.NewChannelHandler(channelSvc).RegisterRoutes(app)

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

	slogx.Info("channel starting on %s", cfg.Address())

	go func() {
		if err := app.Listen(cfg.Address()); err != nil {
			slogx.Fatal("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slogx.Info("channel shutting down")
	app.Shutdown()
}
