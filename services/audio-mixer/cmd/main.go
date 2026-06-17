package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/voxmesh/pkg/config"
	slogx "github.com/voxmesh/pkg/log"
	"github.com/voxmesh/pkg/db"
	voxmqtt "github.com/voxmesh/pkg/mqtt"

	grpcserver "github.com/voxmesh/audio-mixer/internal/grpc"
	"github.com/voxmesh/audio-mixer/internal/pipeline"
)

// noopCodec passes through Opus frames without decode/encode.
// In production (Docker CGO_ENABLED=1), replace with libopus-backed codec.
type noopCodec struct{}

func (n *noopCodec) Decode(opusData []byte, _ int) ([]int16, error) {
	return make([]int16, 0), nil
}

func (n *noopCodec) Encode(pcm []int16, _ int) ([]byte, error) {
	return nil, nil
}

var _ pipeline.Codec = (*noopCodec)(nil)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	redisClient, err := db.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		slogx.Fatal("redis: %v", err)
	}
	defer redisClient.Close()
	_ = redisClient

	// gRPC server for WS Gateway
	grpcSrv := grpcserver.New(cfg.AudioMixerAddr, &noopCodec{})
	go func() {
		if err := grpcSrv.Start(); err != nil {
			slogx.Fatal("gRPC: %v", err)
		}
	}()

	// MQTT subscription for embedded device audio
	mqttCfg := voxmqtt.ClientConfig{
		BrokerURL: cfg.MQTTBrokerURL,
		ClientID:  cfg.MQTTClientID,
		KeepAlive: 30 * time.Second,
	}

	mqttClient, err := voxmqtt.NewClient(mqttCfg,
		func(c mqtt.Client) {
			slogx.Info("[audio-mixer] MQTT connected, subscribing")
		},
		func(c mqtt.Client, err error) {
			slogx.Info("[audio-mixer] MQTT lost: %v", err)
		},
	)
	if err != nil {
		slogx.Info("[audio-mixer] MQTT warning: %v (continuing without MQTT)", err)
	} else {
		defer mqttClient.Close()
	}

	slogx.Info("Audio Mixer started")
	slogx.Info("gRPC: %s, MQTT: %s", cfg.AudioMixerAddr, cfg.MQTTBrokerURL)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slogx.Info("Audio Mixer shutting down")
	grpcSrv.GracefulStop()
}
