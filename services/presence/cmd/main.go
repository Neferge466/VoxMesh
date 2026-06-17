package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voxmesh/pkg/config"
	"github.com/voxmesh/pkg/db"
	slogx "github.com/voxmesh/pkg/log"
	"github.com/voxmesh/pkg/mqtt"

	"github.com/voxmesh/presence/internal/service"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	redisClient, err := db.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		slogx.Fatal("redis: %v", err)
	}
	defer redisClient.Close()

	presenceSvc := service.NewPresenceService(redisClient)

	mqttCfg := mqtt.ClientConfig{
		BrokerURL: cfg.MQTTBrokerURL,
		ClientID:  cfg.MQTTClientID,
		KeepAlive: 30 * time.Second,
	}
	if err := presenceSvc.ConnectMQTT(ctx, mqttCfg); err != nil {
		slogx.Fatal("mqtt: %v", err)
	}
	defer presenceSvc.Close()

	slogx.Info("presence started, client=%s", cfg.MQTTClientID)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slogx.Info("presence shutting down")
}
