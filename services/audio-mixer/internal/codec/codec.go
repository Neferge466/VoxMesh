// Package codec provides Opus audio encode/decode implementations.
//
// Two implementations are provided and selected at compile time via Go build constraints:
//
// Production (CGO_ENABLED=1, Docker):
//   opus_cgo.go  — real libopus via cgo. Requires libopus-dev.
//   Used by Audio Mixer to decode embedded device Opus streams,
//   mix multi-speaker PCM, and re-encode the result.
//
// Development (CGO_ENABLED=0, local dev):
//   opus_noop.go — pass-through that returns empty data without real processing.
//   WHY: libopus is a C library that requires cgo and the opus-dev headers.
//   Installing these on every dev machine is burdensome and unnecessary
//   because the primary audio path (Web → Web) uses WebRTC P2P SRTP —
//   audio frames never reach the Audio Mixer. The mixer only handles
//   the embedded device bridge path (ESP32 → MQTT → Mixer → gRPC → Web),
//   which is not exercised during local frontend development.
//
// Selection is automatic: factory_cgo.go or factory_noop.go provides NewCodec().
package codec
