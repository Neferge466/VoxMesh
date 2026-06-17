package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/voxmesh/pkg/config"
	"github.com/voxmesh/pkg/db"
	slogx "github.com/voxmesh/pkg/log"
	"github.com/voxmesh/pkg/mqtt"

	"github.com/voxmesh/gateway-coordinator/internal/handler"
	"github.com/voxmesh/gateway-coordinator/internal/service"
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

	coordSvc := service.NewGatewayCoordinator(pool, redisClient)

	mqttCfg := mqtt.ClientConfig{
		BrokerURL: cfg.MQTTBrokerURL,
		ClientID:  cfg.MQTTClientID,
		KeepAlive: 30 * time.Second,
	}
	if err := coordSvc.ConnectMQTT(ctx, mqttCfg); err != nil {
		slogx.Fatal("mqtt: %v", err)
	}
	defer coordSvc.Close()

	app := fiber.New(fiber.Config{AppName: "voxmesh-gateway-coordinator"})

	// Prometheus metrics
	fiberprometheus.New("voxmesh-gateway-coordinator").RegisterAt(app, "/metrics")

	handler.NewGatewayHandler(coordSvc).RegisterRoutes(app)

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

	slogx.Info("gateway-coordinator starting on %s", cfg.Address())

	go func() {
		if err := app.Listen(cfg.Address()); err != nil {
			slogx.Fatal("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slogx.Info("gateway-coordinator shutting down")
	app.Shutdown()
}
