package mqtt

import (
	"strings"
	"testing"
)

func TestDeviceTopics(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		contains string
	}{
		{"tx", DeviceAudioTx("esp32_a1"), "voxmesh/devices/esp32_a1/audio/tx/opus"},
		{"rx", DeviceAudioRx("esp32_a1"), "voxmesh/devices/esp32_a1/audio/rx/opus"},
		{"transcript", DeviceTranscript("esp32_a1"), "voxmesh/devices/esp32_a1/command/transcript"},
		{"result", DeviceResult("esp32_a1"), "voxmesh/devices/esp32_a1/command/result"},
		{"status", DeviceStatus("esp32_a1"), "voxmesh/devices/esp32_a1/status"},
		{"config", DeviceConfig("esp32_a1"), "voxmesh/devices/esp32_a1/config"},
	}
	for _, tt := range tests {
		if tt.got != tt.contains {
			t.Errorf("%s: got %s", tt.name, tt.got)
		}
	}
}

func TestChannelTopics(t *testing.T) {
	if got := ChannelDeviceAudio("ch_x", "dev_y"); !strings.Contains(got, "channels/ch_x/audio/device/dev_y/opus") {
		t.Errorf("ChannelDeviceAudio: %s", got)
	}
	if got := ChannelMixedAudio("ch_x"); got != "voxmesh/channels/ch_x/audio/mixed/opus" {
		t.Errorf("ChannelMixedAudio: %s", got)
	}
	if got := ChannelPresence("ch_x"); got != "voxmesh/channels/ch_x/presence" {
		t.Errorf("ChannelPresence: %s", got)
	}
	if got := ChannelState("ch_x"); got != "voxmesh/channels/ch_x/state" {
		t.Errorf("ChannelState: %s", got)
	}
	if got := ChannelControl("ch_x"); got != "voxmesh/channels/ch_x/control" {
		t.Errorf("ChannelControl: %s", got)
	}
}

func TestGatewayTopics(t *testing.T) {
	if got := GatewayRegister("gw_01"); got != "voxmesh/gateways/gw_01/register" {
		t.Errorf("GatewayRegister: %s", got)
	}
	if got := GatewayHeartbeat("gw_01"); got != "voxmesh/gateways/gw_01/heartbeat" {
		t.Errorf("GatewayHeartbeat: %s", got)
	}
	if got := GatewayCommand("gw_01"); got != "voxmesh/gateways/gw_01/command" {
		t.Errorf("GatewayCommand: %s", got)
	}
	if got := GatewayTopology("gw_01"); got != "voxmesh/gateways/gw_01/topology" {
		t.Errorf("GatewayTopology: %s", got)
	}
	if got := GatewayStatus("gw_01"); got != "voxmesh/gateways/gw_01/status" {
		t.Errorf("GatewayStatus: %s", got)
	}
}

func TestPresenceTopics(t *testing.T) {
	if got := PresenceStatus("usr_01"); got != "voxmesh/presence/usr_01/status" {
		t.Errorf("PresenceStatus: %s", got)
	}
	if got := PresenceWildcard(); got != "voxmesh/presence/+/status" {
		t.Errorf("PresenceWildcard: %s", got)
	}
}

func TestSystemTopics(t *testing.T) {
	if got := SystemBroadcast(); got != "voxmesh/system/broadcast" {
		t.Errorf("SystemBroadcast: %s", got)
	}
	if got := SystemVersion(); got != "voxmesh/system/version" {
		t.Errorf("SystemVersion: %s", got)
	}
}

func TestRootTopicPrefix(t *testing.T) {
	topics := []string{
		DeviceAudioTx("d1"),
		ChannelPresence("c1"),
		GatewayHeartbeat("g1"),
		SystemBroadcast(),
	}
	for _, topic := range topics {
		if !strings.HasPrefix(topic, RootTopic) {
			t.Errorf("topic %s should start with %s", topic, RootTopic)
		}
	}
}
