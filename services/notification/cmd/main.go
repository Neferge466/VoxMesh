package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voxmesh/pkg/config"
	slogx "github.com/voxmesh/pkg/log"
	"github.com/voxmesh/pkg/mqtt"

	"github.com/voxmesh/notification/internal/service"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	notifSvc := service.NewNotificationService()

	mqttCfg := mqtt.ClientConfig{
		BrokerURL: cfg.MQTTBrokerURL,
		ClientID:  cfg.MQTTClientID,
		KeepAlive: 30 * time.Second,
	}
	if err := notifSvc.ConnectMQTT(ctx, mqttCfg); err != nil {
		slogx.Fatal("mqtt: %v", err)
	}
	defer notifSvc.Close()

	slogx.Info("Notification Service started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slogx.Info("Notification Service shutting down")
}
