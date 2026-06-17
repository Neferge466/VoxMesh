package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	voxmqtt "github.com/voxmesh/pkg/mqtt"
	"github.com/voxmesh/pkg/model"

	"github.com/voxmesh/gateway-coordinator/internal/repository"
)

const heartbeatTTL = 15 * time.Second
const offlineThreshold = 30 * time.Second

type GatewayCoordinator struct {
	repo    *repository.GatewayRepo
	redis   *redis.Client
	mqttCli *voxmqtt.Client
	mu      sync.Mutex
}

func NewGatewayCoordinator(pool *pgxpool.Pool, redisClient *redis.Client) *GatewayCoordinator {
	return &GatewayCoordinator{
		repo:  repository.NewGatewayRepo(pool),
		redis: redisClient,
	}
}

func (g *GatewayCoordinator) ConnectMQTT(ctx context.Context, cfg voxmqtt.ClientConfig) error {
	client, err := voxmqtt.NewClient(cfg,
		func(c mqtt.Client) {
			log.Println("[gateway-coord] MQTT connected")
			g.subscribeTopics()
		},
		func(c mqtt.Client, err error) {
			log.Printf("[gateway-coord] MQTT lost: %v", err)
		},
	)
	if err != nil {
		return err
	}
	g.mqttCli = client
	g.subscribeTopics()
	go g.monitorLoop(ctx)
	return nil
}

func (g *GatewayCoordinator) subscribeTopics() {
	g.mqttCli.Subscribe(voxmqtt.GatewayRegisterWildcard(), 1, g.handleRegister)
	g.mqttCli.Subscribe(voxmqtt.GatewayHeartbeatWildcard(), 1, g.handleHeartbeat)
}

func (g *GatewayCoordinator) handleRegister(_ mqtt.Client, msg mqtt.Message) {
	var payload voxmqtt.GatewayRegisterPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		return
	}

	ctx := context.Background()
	_, err := g.repo.FindByID(ctx, payload.GatewayID)
	if err != nil {
		// Create if not exists
		g.repo.Create(ctx, &model.Gateway{
			ID:       payload.GatewayID,
			Name:     payload.GatewayID,
			Status:   "online",
			IPAddress: payload.IPAddress,
			Version:  payload.Version,
			Capabilities: model.GatewayCapabilities{
				MaxMeshDevices:   payload.Capabilities.MaxMeshDevices,
				SupportedCodecs:  payload.Capabilities.SupportedCodecs,
				SampleRates:      payload.Capabilities.SampleRates,
				ESPNowEnabled:    payload.Capabilities.ESPNowEnabled,
				WiFiBackhaul:     payload.Capabilities.WiFiBackhaul,
				EthernetBackhaul: payload.Capabilities.EthernetBackhaul,
				MaxESPNowPeers:   payload.Capabilities.MaxESPNowPeers,
			},
		})
	} else {
		g.repo.UpdateStatus(ctx, payload.GatewayID, "online")
	}
	log.Printf("[gateway-coord] gateway %s registered", payload.GatewayID)
}

func (g *GatewayCoordinator) handleHeartbeat(_ mqtt.Client, msg mqtt.Message) {
	var payload voxmqtt.GatewayHeartbeatPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		return
	}

	ctx := context.Background()
	key := "gateway_heartbeat:" + payload.GatewayID
	g.redis.Set(ctx, key, payload.GatewayID, heartbeatTTL)

	// Update last heartbeat
	g.repo.UpdateHeartbeat(ctx, payload.GatewayID)
}

func (g *GatewayCoordinator) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.checkHeartbeats(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (g *GatewayCoordinator) checkHeartbeats(ctx context.Context) {
	gwList, _ := g.repo.List(ctx)
	for _, gw := range gwList {
		key := "gateway_heartbeat:" + gw.ID
		exists, _ := g.redis.Exists(ctx, key).Result()
		if exists == 0 {
			if gw.Status == "online" {
				g.repo.UpdateStatus(ctx, gw.ID, "degraded")
				log.Printf("[gateway-coord] %s DEGRADED (heartbeat lost)", gw.ID)
			} else if gw.Status == "degraded" && gw.LastHeartbeatAt != nil {
				if time.Since(*gw.LastHeartbeatAt) > offlineThreshold {
					g.repo.UpdateStatus(ctx, gw.ID, "offline")
					log.Printf("[gateway-coord] %s OFFLINE", gw.ID)
				}
			}
		}
	}
}

func (g *GatewayCoordinator) GetGateway(ctx context.Context, id string) (*model.Gateway, error) {
	return g.repo.FindByID(ctx, id)
}

func (g *GatewayCoordinator) CreateGateway(ctx context.Context, req model.CreateGatewayRequest) (*model.Gateway, error) {
	gw := &model.Gateway{
		ID:      req.Name,
		Name:    req.Name,
		Status:  "registered",
	}
	if err := g.repo.Create(ctx, gw); err != nil {
		return nil, err
	}
	return gw, nil
}

func (g *GatewayCoordinator) DeleteGateway(ctx context.Context, id string) error {
	return g.repo.Delete(ctx, id)
}

func (g *GatewayCoordinator) GetGateways(ctx context.Context) ([]*model.Gateway, error) {
	return g.repo.List(ctx)
}

func (g *GatewayCoordinator) SendCommand(ctx context.Context, gwID, command string, params map[string]any) error {
	payload := voxmqtt.GatewayCommandPayload{
		CommandID:   randomID(),
		Command:     command,
		Params:      params,
		TimestampMs: time.Now().UnixMilli(),
	}
	return g.mqttCli.Publish(voxmqtt.GatewayCommand(gwID), 1, false, voxmqtt.ToJSON(payload))
}

func (g *GatewayCoordinator) Close() {
	if g.mqttCli != nil {
		g.mqttCli.Close()
	}
}

func randomID() string {
	return fmt.Sprintf("cmd_%d", time.Now().UnixNano()%1000000)
}
