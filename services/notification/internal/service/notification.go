package service

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	voxmqtt "github.com/voxmesh/pkg/mqtt"
)

type NotificationService struct {
	mqttCli  *voxmqtt.Client
	mu       sync.Mutex
	lastSent map[string]time.Time
}

func NewNotificationService() *NotificationService {
	s := &NotificationService{
		lastSent: make(map[string]time.Time),
	}
	go s.cleanupLoop()
	return s
}

func (s *NotificationService) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-5 * time.Minute)
		for k, v := range s.lastSent {
			if v.Before(cutoff) {
				delete(s.lastSent, k)
			}
		}
		s.mu.Unlock()
	}
}

func (s *NotificationService) ConnectMQTT(ctx context.Context, cfg voxmqtt.ClientConfig) error {
	client, err := voxmqtt.NewClient(cfg,
		func(c mqtt.Client) {
			log.Println("[notification] MQTT connected")
			s.subscribeTopics()
		},
		func(c mqtt.Client, err error) {
			log.Printf("[notification] MQTT lost: %v", err)
		},
	)
	if err != nil {
		return err
	}
	s.mqttCli = client
	return s.subscribeTopics()
}

func (s *NotificationService) subscribeTopics() error {
	s.mqttCli.Subscribe(voxmqtt.PresenceWildcard(), 1, s.handlePresenceEvent)
	s.mqttCli.Subscribe(voxmqtt.SystemBroadcast(), 1, s.handleBroadcast)
	return nil
}

func (s *NotificationService) handlePresenceEvent(_ mqtt.Client, msg mqtt.Message) {
	var payload voxmqtt.DeviceStatusPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		return
	}
	state := "online"
	if !payload.Online {
		state = "offline"
	}
	log.Printf("[notification] device %s is %s", payload.DeviceID, state)
}

func (s *NotificationService) handleBroadcast(_ mqtt.Client, msg mqtt.Message) {
	var payload voxmqtt.SystemBroadcastPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		return
	}
	log.Printf("[notification] broadcast [%s]: %s", payload.Severity, payload.Message)
}

// RateLimit checks if a notification can be sent for the given key.
func (s *NotificationService) RateLimit(key string, minInterval time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if last, ok := s.lastSent[key]; ok {
		if time.Since(last) < minInterval {
			return false
		}
	}
	s.lastSent[key] = time.Now()
	return true
}

func (s *NotificationService) Close() {
	if s.mqttCli != nil {
		s.mqttCli.Close()
	}
}
