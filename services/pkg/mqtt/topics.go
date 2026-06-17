// Package mqtt defines the MQTT topic hierarchy constants and builders.
// This is the single source of truth shared by all VoxMesh services
// and the embedded firmware interface contract.
package mqtt

import "fmt"

// Topic prefix
const RootTopic = "voxmesh"

// ---------------------------------------------------------------------------
// Topic builders — use these instead of raw string formatting
// ---------------------------------------------------------------------------

// Devices
func DeviceAudioTx(deviceID string) string   { return fmt.Sprintf("%s/devices/%s/audio/tx/opus", RootTopic, deviceID) }
func DeviceAudioRx(deviceID string) string   { return fmt.Sprintf("%s/devices/%s/audio/rx/opus", RootTopic, deviceID) }
func DeviceTranscript(deviceID string) string { return fmt.Sprintf("%s/devices/%s/command/transcript", RootTopic, deviceID) }
func DeviceResult(deviceID string) string    { return fmt.Sprintf("%s/devices/%s/command/result", RootTopic, deviceID) }
func DeviceStatus(deviceID string) string    { return fmt.Sprintf("%s/devices/%s/status", RootTopic, deviceID) }
func DeviceConfig(deviceID string) string    { return fmt.Sprintf("%s/devices/%s/config", RootTopic, deviceID) }

// Channels
func ChannelDeviceAudio(channelID, deviceID string) string {
	return fmt.Sprintf("%s/channels/%s/audio/device/%s/opus", RootTopic, channelID, deviceID)
}
func ChannelMixedAudio(channelID string) string { return fmt.Sprintf("%s/channels/%s/audio/mixed/opus", RootTopic, channelID) }
func ChannelPresence(channelID string) string   { return fmt.Sprintf("%s/channels/%s/presence", RootTopic, channelID) }
func ChannelState(channelID string) string      { return fmt.Sprintf("%s/channels/%s/state", RootTopic, channelID) }
func ChannelControl(channelID string) string    { return fmt.Sprintf("%s/channels/%s/control", RootTopic, channelID) }

// Gateways
func GatewayRegister(gwID string) string { return fmt.Sprintf("%s/gateways/%s/register", RootTopic, gwID) }
func GatewayHeartbeat(gwID string) string { return fmt.Sprintf("%s/gateways/%s/heartbeat", RootTopic, gwID) }
func GatewayCommand(gwID string) string  { return fmt.Sprintf("%s/gateways/%s/command", RootTopic, gwID) }
func GatewayTopology(gwID string) string { return fmt.Sprintf("%s/gateways/%s/topology", RootTopic, gwID) }
func GatewayStatus(gwID string) string   { return fmt.Sprintf("%s/gateways/%s/status", RootTopic, gwID) }

// Presence
func PresenceStatus(userID string) string { return fmt.Sprintf("%s/presence/%s/status", RootTopic, userID) }

// System
func SystemBroadcast() string { return RootTopic + "/system/broadcast" }
func SystemVersion() string   { return RootTopic + "/system/version" }
func SystemMetrics() string   { return RootTopic + "/system/metrics" }

// ---------------------------------------------------------------------------
// Wildcard topic patterns (for subscriptions)
// ---------------------------------------------------------------------------

// Shared subscription prefix for EMQX: $share/<group>/
const SharedPrefix = "$share"

// Audio Mixer shared subscriptions
func SharedChannelDeviceAudio(group, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/channels/%s/audio/device/+/opus", SharedPrefix, group, RootTopic, channelID)
}

// Gateway wildcard subscriptions
func GatewayWildcardSub(gwID string) string { return fmt.Sprintf("%s/gateways/%s/+/audio/rx/opus", RootTopic, gwID) }

// Presence wildcard
func PresenceWildcard() string { return RootTopic + "/presence/+/status" }

// Gateway heartbeat wildcard
func GatewayHeartbeatWildcard() string { return RootTopic + "/gateways/+/heartbeat" }
func GatewayRegisterWildcard() string  { return RootTopic + "/gateways/+/register" }

// Device wildcards
func DeviceAudioTxWildcard() string      { return RootTopic + "/devices/+/audio/tx/opus" }
func DeviceTranscriptWildcard() string   { return RootTopic + "/devices/+/command/transcript" }

// Channel wildcards for gateway proxying
func ChannelDeviceAudioWildcard() string { return RootTopic + "/channels/+/audio/device/+/opus" }
func ChannelMixedAudioWildcard() string  { return RootTopic + "/channels/+/audio/mixed/opus" }
