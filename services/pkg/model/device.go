package model

import "time"

type Device struct {
	ID               string            `json:"id"`
	GatewayID        string            `json:"gateway_id"`
	Name             string            `json:"name,omitempty"`
	FirmwareVersion  string            `json:"firmware_version,omitempty"`
	Capabilities     DeviceCapabilities `json:"capabilities"`
	LastStatus       *DeviceStatus     `json:"last_status,omitempty"`
	LastSeenAt       *time.Time        `json:"last_seen_at,omitempty"`
	RegisteredAt     time.Time         `json:"registered_at"`
}

type DeviceCapabilities struct {
	NoiseReduction bool `json:"noise_reduction"`
	VoiceCommand   bool `json:"voice_command"`
	Display        bool `json:"display,omitempty"`
	Keypad         bool `json:"keypad,omitempty"`
}

type DeviceStatus struct {
	Online              bool   `json:"online"`
	ChannelID           string `json:"channel_id,omitempty"`
	Muted               bool   `json:"muted"`
	Deafened            bool   `json:"deafened"`
	BatteryPct          int    `json:"battery_pct"`
	RSSIToGateway       int    `json:"rssi_to_gateway"`
	NoiseReductionActive bool  `json:"noise_reduction_active"`
	VoiceCommandActive  bool   `json:"voice_command_active"`
	Codec               string `json:"codec"`
	SampleRateHz        int    `json:"sample_rate_hz"`
	BitrateBps          int    `json:"bitrate_bps"`
	VolumeLevel         int    `json:"volume_level"`
	FirmwareVersion     string `json:"firmware_version,omitempty"`
}
