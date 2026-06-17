package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/voxmesh/pkg/config"
	slogx "github.com/voxmesh/pkg/log"
	voxmqtt "github.com/voxmesh/pkg/mqtt"

	"github.com/voxmesh/command-handler/internal/parser"
)

var mqttClient *voxmqtt.Client

func main() {
	cfg := config.Load()

	mqttCfg := voxmqtt.ClientConfig{
		BrokerURL: cfg.MQTTBrokerURL,
		ClientID:  cfg.MQTTClientID,
		KeepAlive: 30 * time.Second,
	}

	var err error
	mqttClient, err = voxmqtt.NewClient(mqttCfg,
		func(c mqtt.Client) {
			slogx.Info("[command] MQTT connected")
			c.Subscribe(voxmqtt.DeviceTranscriptWildcard(), 1, handleTranscript)
		},
		func(c mqtt.Client, err error) {
			slogx.Info("[command] MQTT lost: %v", err)
		},
	)
	if err != nil {
		slogx.Fatal("mqtt: %v", err)
	}
	defer mqttClient.Close()

	slogx.Info("Command Handler started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slogx.Info("Command Handler shutting down")
}

func handleTranscript(_ mqtt.Client, msg mqtt.Message) {
	var payload voxmqtt.CommandTranscriptPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		return
	}

	slogx.Info("[command] transcript from %s: %s (confidence: %.2f)",
		payload.DeviceID, payload.Transcript, payload.Confidence)

	cmd := parser.Parse(payload.Transcript)
	if cmd == nil {
		slogx.Info("[command] unrecognized: %s", payload.Transcript)
		return
	}

	slogx.Info("[command] parsed intent: %s params=%v", cmd.Action, cmd.Params)

	resultData, _ := json.Marshal(voxmqtt.CommandResultPayload{
		CommandID:   randomID(),
		DeviceID:    payload.DeviceID,
		Success:     true,
		Action:      cmd.Action,
		Message:     "command received",
		TimestampMs: time.Now().UnixMilli(),
	})

	resultTopic := voxmqtt.DeviceResult(payload.DeviceID)
	if err := mqttClient.Publish(resultTopic, 1, false, resultData); err != nil {
		slogx.Info("[command] publish error: %v", err)
	}
}

func randomID() string {
	return fmt.Sprintf("cmd_%d", time.Now().UnixNano()%1000000)
}
