# VoxMesh — Embedded Device MQTT Interface Contract

> **Status**: v1.0.0 | **Target**: ESP32-S3 firmware | **Protocol**: MQTT 5.0

This document defines the MQTT protocol contract between the VoxMesh server and embedded firmware. It serves as the authoritative integration specification for firmware developers. No firmware implementation details are specified.

---

## 1. Connection Parameters

| Parameter | Value | Notes |
|---|---|---|
| Protocol | MQTT 5.0 | User Properties required for audio metadata |
| Transport | TCP/TLS | Port 8883 |
| TLS | Required | CA certificate pre-loaded in firmware |
| Client ID | `{gateway_id}` | Gateway's unique identifier |
| Username | `{gateway_id}` | Same as Client ID |
| Password | `gwk_xxxxxxxx` | Pre-provisioned API key |
| Keep Alive | 30 seconds | |
| Clean Start | true | |
| Session Expiry | 0 | No persistent session; re-subscribe on reconnect |
| Will Topic | `voxmesh/gateways/{gateway_id}/status` | |
| Will Payload | `{"status":"offline","reason":"connection_lost","timestamp_ms":<now>}` | |
| Will QoS | 1 | |
| Will Retain | true | |

## 2. Topics — Subscribe (server → gateway)

| Topic | QoS | Purpose |
|---|---|---|
| `voxmesh/gateways/{gateway_id}/command` | 1 | Server commands to this gateway |
| `voxmesh/gateways/{gateway_id}/topology` | 1 | Server-assigned mesh topology |
| `voxmesh/devices/+/audio/rx/opus` | 1 | Downlink audio for devices behind this gateway |
| `voxmesh/devices/+/command/result` | 1 | Voice command results for devices |
| `voxmesh/channels/+/audio/mixed/opus` | 0 | Mixed channel audio (optional, monitoring) |
| `voxmesh/system/broadcast` | 1 | System-wide announcements |

## 3. Topics — Publish (gateway → server)

| Topic | QoS | Retain | Purpose |
|---|---|---|---|
| `voxmesh/gateways/{gateway_id}/register` | 1 | No | Registration on connect |
| `voxmesh/gateways/{gateway_id}/heartbeat` | 1 | No | Periodic heartbeat (every 5s) |
| `voxmesh/devices/{device_id}/audio/tx/opus` | 1 | No | Uplink audio from device |
| `voxmesh/devices/{device_id}/status` | 1 | **Yes** | Device online status and telemetry |
| `voxmesh/devices/{device_id}/command/transcript` | 1 | No | Voice command transcription |
| `voxmesh/channels/{channel_id}/control` | 1 | No | PTT/mute/deafen requests on behalf of device |

## 4. Payload Specifications

### 4.1 Gateway Registration
Topic: `voxmesh/gateways/{gateway_id}/register` (QoS 1)

```json
{
  "gateway_id": "gw_01",
  "api_key": "gwk_a1b2c3d4e5f6",
  "capabilities": {
    "max_mesh_devices": 50,
    "supported_codecs": ["opus"],
    "sample_rates": [8000, 16000],
    "espnow_enabled": true,
    "wifi_backhaul": true,
    "ethernet_backhaul": false,
    "max_espnow_peers": 20
  },
  "version": "1.2.3",
  "ip_address": "192.168.1.100"
}
```

### 4.2 Gateway Topology (server → gateway)
Topic: `voxmesh/gateways/{gateway_id}/topology` (QoS 1)

```json
{
  "gateway_id": "gw_01",
  "status": "active",
  "mesh_channel": 6,
  "mesh_pan_id": "0xVOXM",
  "assigned_devices": [
    {
      "device_id": "esp32_a1b2c3d4",
      "mac_addr": "AA:BB:CC:DD:EE:FF",
      "channel_id": "ch_abc123",
      "audio_rx_topic": "voxmesh/devices/esp32_a1b2c3d4/audio/rx/opus",
      "audio_tx_topic": "voxmesh/devices/esp32_a1b2c3d4/audio/tx/opus"
    }
  ],
  "redundant_gateway": "gw_02",
  "mqtt_subscribe_topics": [
    "voxmesh/gateways/gw_01/command",
    "voxmesh/devices/+/audio/rx/opus",
    "voxmesh/system/broadcast"
  ]
}
```

### 4.3 Gateway Heartbeat
Topic: `voxmesh/gateways/{gateway_id}/heartbeat` (QoS 1, every 5s)

```json
{
  "gateway_id": "gw_01",
  "timestamp_ms": 1718234567890,
  "uptime_sec": 86400,
  "cpu_pct": 34.2,
  "memory_free_kb": 51200,
  "wifi_rssi": -55,
  "mesh_devices_count": 12,
  "mesh_devices": [
    {
      "device_id": "esp32_a1b2c3d4",
      "rssi": -42,
      "hops": 1,
      "battery_pct": 85,
      "snr_db": 18.5
    }
  ],
  "version": "1.2.3"
}
```

### 4.4 Device Audio Uplink
Topic: `voxmesh/devices/{device_id}/audio/tx/opus` (QoS 1)

**Payload**: Binary — concatenated Opus frames. No JSON wrapper.

**MQTT5 User Properties** (carried in PUBLISH packet header, NOT in payload):

| Property | Type | Example |
|---|---|---|
| `device_id` | string | `esp32_a1b2c3d4` |
| `channel_id` | string | `ch_abc123` |
| `seq` | uint32 | `1042` |
| `timestamp` | int64 | `1718234567890` |
| `frame_count` | uint8 | `3` |
| `energy` | float32 | `0.024` |
| `sample_rate` | uint16 | `16000` |

Each Opus frame = 20ms audio. With `frame_count=3`, payload = 60ms (3 concatenated frames).

### 4.5 Device Audio Downlink
Topic: `voxmesh/devices/{device_id}/audio/rx/opus` (QoS 1)

Same binary format as uplink. User Properties additionally include:
| `speaker_id` | string | `esp32_x9y8z7` |
| `speaker_display_name` | string | `Bob` |

### 4.6 Device Status
Topic: `voxmesh/devices/{device_id}/status` (QoS 1, RETAINED)

```json
{
  "device_id": "esp32_a1b2c3d4",
  "gateway_id": "gw_01",
  "online": true,
  "channel_id": "ch_abc123",
  "muted": false,
  "deafened": false,
  "battery_pct": 85,
  "battery_voltage_mv": 3950,
  "rssi_to_gateway": -42,
  "noise_reduction_active": true,
  "voice_command_active": true,
  "codec": "opus",
  "sample_rate_hz": 16000,
  "bitrate_bps": 32000,
  "ptt_active": false,
  "volume_level": 75,
  "firmware_version": "0.9.2",
  "timestamp_ms": 1718234567890
}
```

**Publish triggers**:
1. Device connects to mesh
2. Device disconnects from mesh (`online: false`)
3. Every 30 seconds (active keepalive)
4. Immediately on any property change

### 4.7 Voice Command Transcript
Topic: `voxmesh/devices/{device_id}/command/transcript` (QoS 1)

```json
{
  "device_id": "esp32_a1b2c3d4",
  "transcript": "join channel alpha",
  "confidence": 0.92,
  "locale": "en-US",
  "wake_word": "hey voxmesh",
  "wake_word_confidence": 0.98,
  "snr_db": 15.3,
  "audio_duration_ms": 1200,
  "timestamp_ms": 1718234567890
}
```

### 4.8 Voice Command Result
Topic: `voxmesh/devices/{device_id}/command/result` (QoS 1)

```json
{
  "command_id": "cmd_a1b2c3",
  "device_id": "esp32_a1b2c3d4",
  "success": true,
  "action": "join_channel",
  "params": {"channel_id": "ch_alpha", "channel_name": "Alpha Squad"},
  "message": "Joined Alpha Squad",
  "error_code": 0,
  "timestamp_ms": 1718234568200
}
```

### 4.9 Gateway Command
Topic: `voxmesh/gateways/{gateway_id}/command` (QoS 1, server → gateway)

```json
{
  "command_id": "cmd_xyz789",
  "command": "adopt_device",
  "params": {
    "device_id": "esp32_a1b2c3d4",
    "mac_addr": "AA:BB:CC:DD:EE:FF",
    "channel_id": "ch_def456"
  },
  "timestamp_ms": 1718234567890
}
```

**Known commands**:
| Command | Purpose | Key Params |
|---|---|---|
| `adopt_device` | Take over device during failover | device_id, mac_addr, channel_id |
| `release_device` | Stop serving a device | device_id |
| `update_firmware` | OTA firmware update | url, version, sha256_checksum |
| `set_mesh_power` | Adjust ESP-NOW TX power | power_dbm (0–20) |
| `reconnect_mqtt` | Force MQTT reconnection | — |
| `restart` | Reboot gateway | — |
| `set_channel_for_device` | Move device to different channel | device_id, new_channel_id |
| `ping` | Request immediate heartbeat | — |

### 4.10 Channel Control
Topic: `voxmesh/channels/{channel_id}/control` (QoS 1)

```json
{
  "device_id": "esp32_a1b2c3d4",
  "channel_id": "ch_abc123",
  "action": "ptt_start",
  "timestamp_ms": 1718234567890
}
```

**Actions**: `ptt_start`, `ptt_end`, `mute`, `unmute`, `deafen`, `undeafen`, `leave_channel`, `join_channel`

---

## 5. Device Behavior Contract

The following is the expected behavior of embedded devices from the server's perspective — the contract firmware must fulfill.

1. **Audio Encoding**: All audio MUST be Opus-encoded. 20ms frame duration. Sample rates: 8/16/48 kHz. Bitrate: 8–128 kbps.
2. **Frame Transmission**: Batch 3 frames (60ms) per MQTT publish for efficiency. ESP-NOW direct sends single frames (20ms) for lowest latency.
3. **Silence Suppression**: Do NOT publish when silent. Send at most one silence/heartbeat frame every 2 seconds to indicate "connected but silent."
4. **Voice Commands**: Local wake-word detection triggers ASR. Send full transcript only — no raw audio uploads.
5. **Noise Reduction**: Performed locally before Opus encoding. Server receives clean audio.
6. **Status Reporting**: Report battery, RSSI, online status on connect, on change, and at minimum every 30 seconds.
7. **Channel Membership**: One channel per device at a time. Join/leave via voice command, physical button, or server command.
8. **Mesh Routing**: Participate in ESP-NOW multi-hop relay. Devices >2 hops from gateway relay through intermediate nodes.
9. **Reconnection**: On ESP-NOW disconnect, seek alternative gateway. On WiFi loss, continue local mesh only (no cloud bridge).
