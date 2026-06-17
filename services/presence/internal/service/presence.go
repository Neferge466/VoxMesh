package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/redis/go-redis/v9"
	voxmqtt "github.com/voxmesh/pkg/mqtt"
)

const presenceKeyPrefix = "presence:user:"
const presenceTTL = 30 * time.Second

type PresenceService struct {
	redis   *redis.Client
	mqttCli *voxmqtt.Client
}

func NewPresenceService(redisClient *redis.Client) *PresenceService {
	return &PresenceService{redis: redisClient}
}

func (s *PresenceService) ConnectMQTT(ctx context.Context, cfg voxmqtt.ClientConfig) error {
	client, err := voxmqtt.NewClient(cfg,
		func(c mqtt.Client) {
			log.Println("[presence] MQTT connected, re-subscribing")
			s.subscribeTopics()
		},
		func(c mqtt.Client, err error) {
			log.Printf("[presence] MQTT connection lost: %v", err)
		},
	)
	if err != nil {
		return err
	}
	s.mqttCli = client
	return s.subscribeTopics()
}

func (s *PresenceService) subscribeTopics() error {
	if err := s.mqttCli.Subscribe(voxmqtt.PresenceWildcard(), 1, s.handlePresence); err != nil {
		return err
	}
	return nil
}

func (s *PresenceService) handlePresence(_ mqtt.Client, msg mqtt.Message) {
	var payload voxmqtt.DeviceStatusPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("[presence] parse error: %v", err)
		return
	}

	ctx := context.Background()
	key := presenceKeyPrefix + payload.DeviceID

	if !payload.Online {
		s.redis.Del(ctx, key)
		log.Printf("[presence] %s offline", payload.DeviceID)
		return
	}

	data, _ := json.Marshal(map[string]interface{}{
		"user_id":      payload.DeviceID,
		"display_name": payload.DeviceID,
		"channel_id":   payload.ChannelID,
		"muted":        payload.Muted,
		"speaking":     false,
		"client_type":  "embedded",
		"gateway_id":   payload.GatewayID,
	})
	s.redis.Set(ctx, key, data, presenceTTL)
}

func (s *PresenceService) MarkWebUserOnline(ctx context.Context, userID, channelID, displayName string) error {
	key := presenceKeyPrefix + userID
	data, _ := json.Marshal(map[string]interface{}{
		"user_id":      userID,
		"display_name": displayName,
		"channel_id":   channelID,
		"client_type":  "web",
	})
	return s.redis.Set(ctx, key, data, presenceTTL).Err()
}

func (s *PresenceService) MarkOffline(ctx context.Context, userID string) {
	s.redis.Del(ctx, presenceKeyPrefix+userID)
}

func (s *PresenceService) GetChannelPresence(ctx context.Context, channelID string) ([]voxmqtt.PresenceMember, error) {
	var members []voxmqtt.PresenceMember
	iter := s.redis.Scan(ctx, 0, presenceKeyPrefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		data, err := s.redis.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var member voxmqtt.PresenceMember
		if err := json.Unmarshal(data, &member); err != nil {
			continue
		}
		if member.ChannelID == channelID || channelID == "" {
			members = append(members, member)
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return members, nil
}

func (s *PresenceService) RefreshTTL(ctx context.Context, userID string) error {
	key := presenceKeyPrefix + userID
	return s.redis.Expire(ctx, key, presenceTTL).Err()
}

func (s *PresenceService) Close() {
	if s.mqttCli != nil {
		s.mqttCli.Close()
	}
}
