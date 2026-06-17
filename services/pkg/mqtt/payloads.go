// Package mqtt defines JSON payload structures for MQTT messages.
package mqtt

import "encoding/json"

// ---------------------------------------------------------------------------
// Device payloads
// ---------------------------------------------------------------------------

type DeviceStatusPayload struct {
	DeviceID            string `json:"device_id"`
	GatewayID           string `json:"gateway_id"`
	Online              bool   `json:"online"`
	Muted               bool   `json:"muted"`
	Deafened            bool   `json:"deafened"`
	ChannelID           string `json:"channel_id"`
	BatteryPct          int    `json:"battery_pct"`
	BatteryVoltageMV    int    `json:"battery_voltage_mv,omitempty"`
	RSSIToGateway       int    `json:"rssi_to_gateway"`
	NoiseReductionActive bool  `json:"noise_reduction_active"`
	VoiceCommandActive  bool   `json:"voice_command_active"`
	Codec               string `json:"codec"`
	SampleRateHz        int    `json:"sample_rate_hz"`
	BitrateBps          int    `json:"bitrate_bps"`
	PTTActive           bool   `json:"ptt_active"`
	VADThresholdDB      int    `json:"vad_threshold_db,omitempty"`
	VolumeLevel         int    `json:"volume_level"`
	FirmwareVersion     string `json:"firmware_version,omitempty"`
	TimestampMs         int64  `json:"timestamp_ms"`
}

// ---------------------------------------------------------------------------
// Channel payloads
// ---------------------------------------------------------------------------

type ChannelPresencePayload struct {
	ChannelID   string          `json:"channel_id"`
	TimestampMs int64           `json:"timestamp_ms"`
	Members     []PresenceMember `json:"members"`
}

type PresenceMember struct {
	UserID      string `json:"user_id"`
	DeviceID    string `json:"device_id,omitempty"`
	DisplayName string `json:"display_name"`
	ChannelID   string `json:"channel_id,omitempty"`
	ClientType  string `json:"client_type"` // "web" | "embedded"
	Speaking    bool   `json:"speaking"`
	Muted       bool   `json:"muted"`
	Deafened    bool   `json:"deafened"`
	GatewayID   string `json:"gateway_id,omitempty"`
}

type ChannelControlPayload struct {
	DeviceID    string `json:"device_id"`
	ChannelID   string `json:"channel_id"`
	Action      string `json:"action"` // ptt_start, ptt_end, mute, unmute, etc.
	TimestampMs int64  `json:"timestamp_ms"`
}

// ---------------------------------------------------------------------------
// Gateway payloads
// ---------------------------------------------------------------------------

type GatewayRegisterPayload struct {
	GatewayID    string            `json:"gateway_id"`
	APIKey       string            `json:"api_key"`
	Capabilities GatewayCapabilities `json:"capabilities"`
	Version      string            `json:"version"`
	IPAddress    string            `json:"ip_address"`
}

type GatewayCapabilities struct {
	MaxMeshDevices    int      `json:"max_mesh_devices"`
	SupportedCodecs   []string `json:"supported_codecs"`
	SampleRates       []int    `json:"sample_rates"`
	ESPNowEnabled     bool     `json:"espnow_enabled"`
	WiFiBackhaul      bool     `json:"wifi_backhaul"`
	EthernetBackhaul  bool     `json:"ethernet_backhaul"`
	MaxESPNowPeers    int      `json:"max_espnow_peers,omitempty"`
}

type GatewayHeartbeatPayload struct {
	GatewayID        string             `json:"gateway_id"`
	TimestampMs      int64              `json:"timestamp_ms"`
	UptimeSec        int64              `json:"uptime_sec"`
	CPUPct           float64            `json:"cpu_pct,omitempty"`
	MemoryFreeKB     int64              `json:"memory_free_kb,omitempty"`
	WiFiRSSI         int                `json:"wifi_rssi"`
	MeshDevicesCount int                `json:"mesh_devices_count"`
	MeshDevices      []MeshDeviceInfo   `json:"mesh_devices"`
	Version          string             `json:"version"`
}

type MeshDeviceInfo struct {
	DeviceID   string `json:"device_id"`
	RSSI       int    `json:"rssi"`
	Hops       int    `json:"hops"`
	BatteryPct int    `json:"battery_pct,omitempty"`
	SNRDB      float64 `json:"snr_db,omitempty"`
}

type GatewayTopologyPayload struct {
	GatewayID        string              `json:"gateway_id"`
	Status           string              `json:"status"`
	MeshChannel      int                 `json:"mesh_channel,omitempty"`
	MeshPANID        string              `json:"mesh_pan_id,omitempty"`
	AssignedDevices  []AssignedDevice    `json:"assigned_devices"`
	RedundantGateway string              `json:"redundant_gateway,omitempty"`
	MqttTopics       []string            `json:"mqtt_subscribe_topics"`
}

type AssignedDevice struct {
	DeviceID    string `json:"device_id"`
	MACAddr     string `json:"mac_addr"`
	ChannelID   string `json:"channel_id"`
	AudioRxTopic string `json:"audio_rx_topic"`
	AudioTxTopic string `json:"audio_tx_topic"`
}

type GatewayCommandPayload struct {
	CommandID   string         `json:"command_id"`
	Command     string         `json:"command"`
	Params      map[string]any `json:"params"`
	TimestampMs int64          `json:"timestamp_ms"`
}

type GatewayStatusPayload struct {
	GatewayID  string `json:"gateway_id"`
	Status     string `json:"status"` // online, degraded, offline
	Reason     string `json:"reason,omitempty"`
	TimestampMs int64 `json:"timestamp_ms"`
}

// ---------------------------------------------------------------------------
// Voice command payloads
// ---------------------------------------------------------------------------

type CommandTranscriptPayload struct {
	DeviceID            string  `json:"device_id"`
	Transcript          string  `json:"transcript"`
	Confidence          float64 `json:"confidence"`
	Locale              string  `json:"locale"`
	WakeWord            string  `json:"wake_word,omitempty"`
	WakeWordConfidence  float64 `json:"wake_word_confidence,omitempty"`
	SNRDB               float64 `json:"snr_db"`
	AudioDurationMs     int     `json:"audio_duration_ms"`
	TimestampMs         int64   `json:"timestamp_ms"`
}

type CommandResultPayload struct {
	CommandID   string `json:"command_id"`
	DeviceID    string `json:"device_id"`
	Success     bool   `json:"success"`
	Action      string `json:"action"`
	Params      map[string]string `json:"params,omitempty"`
	Message     string `json:"message"`
	ErrorCode   int    `json:"error_code"`
	TimestampMs int64  `json:"timestamp_ms"`
}

// ---------------------------------------------------------------------------
// System payloads
// ---------------------------------------------------------------------------

type SystemBroadcastPayload struct {
	Severity    string `json:"severity"` // info, warning, critical
	Message     string `json:"message"`
	TimestampMs int64  `json:"timestamp_ms"`
}

type SystemVersionPayload struct {
	Version          string `json:"version"`
	MinClientVersion string `json:"min_client_version"`
	APIVersion       string `json:"api_version"`
}

// ---------------------------------------------------------------------------
// MQTT5 User Properties (audio frame metadata)
// ---------------------------------------------------------------------------

type AudioFrameMetadata struct {
	DeviceID    string  `json:"device_id,omitempty"`
	ChannelID   string  `json:"channel_id,omitempty"`
	Seq         uint32  `json:"seq"`
	Timestamp   int64   `json:"timestamp"`
	FrameCount  uint8   `json:"frame_count"`
	Energy      float32 `json:"energy"`
	SampleRate  uint16  `json:"sample_rate,omitempty"`
	SpeakerID   string  `json:"speaker_id,omitempty"`
}

// ToJSON serializes any payload. Panics on error (should never happen with these structs).
func ToJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic("mqtt payload marshal: " + err.Error())
	}
	return b
}

// FromJSON deserializes a payload.
func FromJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
