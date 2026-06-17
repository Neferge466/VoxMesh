package model

import "time"

type Gateway struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Status           string            `json:"status"` // online, degraded, offline
	IPAddress        string            `json:"ip_address,omitempty"`
	Version          string            `json:"version,omitempty"`
	Capabilities     GatewayCapabilities `json:"capabilities"`
	LastHeartbeatAt  *time.Time        `json:"last_heartbeat_at,omitempty"`
	RegisteredAt     time.Time         `json:"registered_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	// Computed
	DeviceCount int `json:"device_count,omitempty"`
}

type GatewayCapabilities struct {
	MaxMeshDevices   int      `json:"max_mesh_devices"`
	SupportedCodecs  []string `json:"supported_codecs"`
	SampleRates      []int    `json:"sample_rates"`
	ESPNowEnabled    bool     `json:"espnow_enabled"`
	WiFiBackhaul     bool     `json:"wifi_backhaul"`
	EthernetBackhaul bool     `json:"ethernet_backhaul"`
	MaxESPNowPeers   int      `json:"max_espnow_peers,omitempty"`
}

type CreateGatewayRequest struct {
	Name string `json:"name"`
	APIKey string `json:"api_key"`
}

type GatewayCommandRequest struct {
	Command string         `json:"command"`
	Params  map[string]any `json:"params,omitempty"`
}

type GatewayCommandResponse struct {
	CommandID string `json:"command_id"`
}
